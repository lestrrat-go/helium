<?xml version="1.0" encoding="utf-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
   version="3.0"
   xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:mf="http://example.com/mf"
   exclude-result-prefixes="#all"
   expand-text="yes">
   
   <xsl:function name="mf:wrap-if-populated" as="element()" visibility="public">
      <xsl:param name="items" as="item()*"/>
      <xsl:param name="wrapper-name" as="xs:QName"/>
      <xsl:where-populated>
         <xsl:element name="{$wrapper-name}" namespace="{namespace-uri-from-QName($wrapper-name)}">
            <xsl:apply-templates select="$items"/>
         </xsl:element>      
      </xsl:where-populated>
   </xsl:function>
   
   <xsl:template match="list[@type = 'ordered']">
      <xsl:sequence select="mf:wrap-if-populated(item, QName('', 'ol'))"/>
   </xsl:template>
   
   <xsl:template match="list">
      <xsl:sequence select="mf:wrap-if-populated(item, QName('', 'ul'))"/>
   </xsl:template>
   
   <xsl:template match="list/item">
      <li>
         <xsl:apply-templates/>
      </li>
   </xsl:template>
   
   <xsl:template match="root">
      <html>
         <head>
            <title>Example</title>
         </head>
         <body>
            <xsl:apply-templates/>
         </body>
      </html>
   </xsl:template>
   
   <xsl:output method="xml" indent="yes"/>
   
   <xsl:mode on-no-match="shallow-copy"/>
   
   <xsl:template match="/" name="xsl:initial-template">
      <xsl:next-match/>
      <xsl:comment xmlns:saxon="http://saxon.sf.net/">Run with {system-property('xsl:product-name')} {system-property('xsl:product-version')} {system-property('Q{http://saxon.sf.net/}platform')}</xsl:comment>
   </xsl:template>
   
</xsl:stylesheet>