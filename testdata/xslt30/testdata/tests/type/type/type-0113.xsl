<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema">

  <!-- PURPOSE: Test whitespace facet on casting from string. -->
  <?spec fo#casting-from-strings?>

  <xsl:template match="/">
    <out>;
      <f><xsl:value-of select="xs:boolean(' 0 ')"/></f>;
      <t><xsl:value-of select="xs:boolean(' true&#xd;')"/></t>;
      <t><xsl:value-of select="xs:decimal('&#x9; &#x9;12.45&#x9; &#x9;')"/></t>;
      <f><xsl:value-of select="xs:anyURI('   http://example.com/  ')"/></f>;
      <f val="c &#9; d"><xsl:value-of select="xs:string(' a  b c &#xd; d  e f ')"/></f>;
      <t><xsl:value-of select="xs:normalizedString(' a  b c &#x9; d  e f ')"/></t>;
      <t><xsl:value-of select="xs:NMTOKEN(' 1234a   &#x9;  ')"/></t>;
      <t><xsl:value-of select="xs:ID('  c1234 &#xa;  ')"/></t>;
    </out>
  </xsl:template>

</xsl:stylesheet>