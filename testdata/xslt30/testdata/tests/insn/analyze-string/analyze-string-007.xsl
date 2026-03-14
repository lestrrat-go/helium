<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

 <?spec xslt#analyze-string?>
  <!-- PURPOSE: test xsl:analyze-string using multi-line mode. -->
  
  <xsl:template match="/">
    <out>
      <xsl:analyze-string select="doc" regex="^.+$" flags="m">
       <xsl:matching-substring><p><xsl:value-of select="."/></p></xsl:matching-substring>
      </xsl:analyze-string>      
    </out>
  </xsl:template>
  
  <!-- orginal regex was ^.*$ but that fails on some platforms because it's a regular expression
       that matches a zero-length string -->

</xsl:stylesheet>