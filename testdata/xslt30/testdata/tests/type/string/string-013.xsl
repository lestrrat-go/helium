<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-concat?>
<!-- PURPOSE:   Test of 'concat()' function. -->
<xsl:template match="doc">
   <out>
      <xsl:value-of select='concat("x","yz")'/>
   </out>
</xsl:template>
 
</xsl:stylesheet>
