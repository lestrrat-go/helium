<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">


  <!-- PURPOSE: ERR XTSE1130 - neither matching-substring nor non-matching-substring present. -->
  
  <xsl:template match="/">
    <out>;
      <xsl:analyze-string select="doc" regex="e">
       
      </xsl:analyze-string>      
    </out>
  </xsl:template>

</xsl:stylesheet>