<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-substring-after?>
<!-- PURPOSE:   Test of 'substring-after()' function. -->
<xsl:template match="/">
   <out>
      <xsl:value-of select="substring-after('1999/04/01', '/')"/>
   </out>
</xsl:template>
 
</xsl:stylesheet>
