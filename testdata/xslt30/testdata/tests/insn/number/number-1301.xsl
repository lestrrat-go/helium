<?xml version="1.0" encoding="UTF-8"?>
<?spec xslt#number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
xmlns:bib="http://purl.org/NET/xbiblio/citeproc"
  exclude-result-prefixes="xs bib" version="2.0">

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
    <number>
      <xsl:value-of select="bib:footcite(.)"/>
    </number>
  </xsl:template>

  <xsl:template match="citation[ancestor::footnote]">
    <nonumber/>
  </xsl:template>

  <xsl:function name="bib:footcite" as="xs:string">
    <xsl:param name="footciteable" as="element()"/>
      <xsl:number level="any" select="$footciteable"
count="footnote|citation[not(ancestor::footnote)]"/>
  </xsl:function>

</xsl:stylesheet>
