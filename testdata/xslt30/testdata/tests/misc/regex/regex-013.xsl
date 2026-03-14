<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">


  <!-- PURPOSE: test tokenize() with a pattern that matches ambiguously. -->
  
  <xsl:template match="/">
    <out>
        <a><xsl:value-of select='tokenize("abracadabra", "ab|a")' separator="|"/></a>
        <b><xsl:value-of select='tokenize("abracadabra", "a|ab")' separator="|"/></b>
    </out>
  </xsl:template>

</xsl:stylesheet>