<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">


  <!-- PURPOSE: test xf:tokenize() function. -->
  <?spec fo#func-tokenize?>
 

  <xsl:template match="/">
    <out>[<xsl:value-of select="tokenize(doc, '\s+')" separator="|"/>]</out>
  </xsl:template>

</xsl:stylesheet>