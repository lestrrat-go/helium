<?xml version="1.0" encoding="UTF-8"?>
  <?spec xslt#number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:bib="http://purl.org/NET/xbiblio/citeproc"
    exclude-result-prefixes="xs bib" version="2.0">
    
    <xsl:key name="refs" match="citation" use="@href"/>
    
    <xsl:template match="doc">
      <list>
        <xsl:apply-templates
          select="//footnote|//citation[not(ancestor::footnote)]"/>
      </list>
    </xsl:template>
    
    <xsl:template match="p">
      <xsl:apply-templates/>
    </xsl:template>
    
    <xsl:template match="footnote">
      <number>
        <xsl:value-of select="bib:footcite(.)"/>
        <xsl:apply-templates select="citation"/>
      </number>
    </xsl:template>
    
    <xsl:template match="citation">
      <xsl:variable name="first">
        <xsl:value-of select="bib:first-reference(.)"/>
      </xsl:variable>
      <number first="{$first}">
        <xsl:value-of select="bib:footcite(.)"/>
      </number>
    </xsl:template>
    
    <xsl:template match="citation[ancestor::footnote]">
      <xsl:variable name="first">
        <xsl:value-of select="bib:first-reference(.)"/>
      </xsl:variable>
      <nonumber first="{$first}"/>
    </xsl:template>
    
    <xsl:function name="bib:footcite" as="xs:string">
      <xsl:param name="footciteable" as="element()"/>
      <xsl:for-each select="$footciteable">
        <xsl:number level="any" select="."
          count="footnote|citation[not(ancestor::footnote)]"/>
      </xsl:for-each>
    </xsl:function>
    
    <xsl:function name="bib:first-reference" as="xs:boolean">
      <xsl:param name="cite-ref" as="node()"/>
      <!--xsl:message> first-ref: cite-ref = <xsl:value-of select="generate-id($cite-ref)"/>, 
                          href = <xsl:value-of select="$cite-ref/@href"/>,
                          first-with-key = <xsl:value-of select="generate-id(key('refs', $cite-ref/@href, $cite-ref)[1])"/></xsl:message-->
      <xsl:sequence select="$cite-ref is key('refs', $cite-ref/@href, $cite-ref)[1]"/>
    </xsl:function>
    
  </xsl:stylesheet>