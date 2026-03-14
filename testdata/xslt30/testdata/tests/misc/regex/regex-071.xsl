<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
  
  
  <!-- PURPOSE: test replace() with special characters in replacement string -->
  
  <xsl:template match="/" name="main">
    <xsl:param name="in" select="'Replace some WORDS in this SENTENCE.'"/>
    <out>
      <A><xsl:value-of select="replace($in, '[A-Z][A-Z]+', '$0$0')"/></A>
      <B><xsl:value-of select="replace($in, '[A-Z][A-Z]+', '\$\$')"/></B>
      <C><xsl:value-of select="replace($in, '[A-Z][A-Z]+', '$0-$0')"/></C>
      <D><xsl:value-of select="replace($in, '[A-Z][A-Z]+', '\$-\$')"/></D>
    </out>
  </xsl:template>
  
</xsl:stylesheet>