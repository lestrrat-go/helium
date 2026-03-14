<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- PURPOSE: regex bug sumbitted by Bob du Charme. -->
  
  <xsl:template match="/">
  <o>
    <b><xsl:value-of select="matches('AAB', 'A{2}')"/></b>
    <xsl:analyze-string select="'AAB'"
                         regex="A{2}">
      <xsl:matching-substring>
         <m><xsl:value-of select="."/></m>
      </xsl:matching-substring>
      <xsl:non-matching-substring>
          <n><xsl:value-of select="."/></n>
      </xsl:non-matching-substring>
    </xsl:analyze-string>
  </o>  
  </xsl:template>

</xsl:stylesheet>