<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
   version="3.0"
   xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:mf="http://example.com/mf"
   xmlns:ex="http://example.com/ex"
   exclude-result-prefixes="#all"
   expand-text="yes">
   
   <xsl:function name="mf:group-chunks" as="item()*" streamability="absorbing">
      <xsl:param name="items" as="item()*"/>
      <xsl:param name="chunk-size" as="xs:integer"/>
      <!--<xsl:param name="handle-chunk" as="function(item()*) as item()*"/>-->
      <xsl:for-each-group select="$items" group-adjacent="(position() - 1) idiv $chunk-size">
         <xsl:sequence select="ex:wrap-rows(current-group(), QName((), 'chunk'))"/>
      </xsl:for-each-group>
   </xsl:function>
   
   <xsl:function name="ex:wrap-rows" as="element()" streamability="absorbing">
      <xsl:param name="group" as="element()*"/>
      <xsl:param name="wrapper-name" as="xs:QName"/>
      <xsl:element name="{$wrapper-name}" namespace="{namespace-uri-from-QName($wrapper-name)}">
         <xsl:sequence select="$group"/>
      </xsl:element>
   </xsl:function>
   
   <xsl:output method="xml" indent="yes"/>
   
   <xsl:mode streamable="yes"/>
   
   <xsl:template match="root">
      <xsl:copy>
         <xsl:sequence select="mf:group-chunks(item, 3 (:, ex:wrap-rows(?, QName((), 'chunk')):) )"/>      
      </xsl:copy>
   </xsl:template>
   
   <xsl:template match="/">
      <xsl:next-match/>
      <xsl:comment xmlns:saxon="http://saxon.sf.net/">Run with {system-property('xsl:product-name')} {system-property('xsl:product-version')} {system-property('Q{http://saxon.sf.net/}platform')}</xsl:comment>
   </xsl:template>
   
</xsl:stylesheet>