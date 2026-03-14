<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">


  <!-- PURPOSE: test regex containing \s in conjunction with 'x' flag (Priscilla Walmsley bug). -->
  <?spec fo#func-matches?>
  
  <xsl:output encoding="iso-8859-1"/>
  
  <xsl:template match="/">
    <out>
     <a><xsl:value-of select='matches("abc", "a\sbc", "x")'/></a>
     <a><xsl:value-of select='matches("a bc", "a\sbc", "x")'/></a>
    </out>
  </xsl:template>

</xsl:stylesheet>