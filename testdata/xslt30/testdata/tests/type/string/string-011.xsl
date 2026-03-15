<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-normalize-space?>
<!-- PURPOSE:   Test of 'normalize-space()' function. -->
<xsl:template match="/doc">
   <out>
      <xsl:value-of select="normalize-space('&#9;&#10;&#13; ab    cd&#9;&#10;&#13;ef&#9;&#10;&#13; ')"/>
   </out>
</xsl:template>
 
</xsl:stylesheet>
