<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-contains?>
<!-- PURPOSE:   Test of 'contains()' function. -->
<xsl:template match="/">
   <out>
      <xsl:value-of select="contains('ENCYCLOPEDIA', 'CYCL')"/>
   </out>
</xsl:template>
 
</xsl:stylesheet>
