<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">


<?spec fo#func-substring?>
<!-- PURPOSE:   Test of 'substring()' function. -->
<xsl:template match="/">
   <out>
      <xsl:value-of select="substring('1999/04/01', 1, 4)"/>
   </out>
</xsl:template>
 
</xsl:stylesheet>
