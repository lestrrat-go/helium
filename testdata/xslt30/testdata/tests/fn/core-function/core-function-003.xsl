<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-concat?>
  <!-- Purpose: Test of 'concat()' function with 'concat()' as an argument.
       Only the first argument of the outer "concat" is build by
       a call to another "concat" function. -->

<xsl:template match="doc">
 <out>
   <xsl:value-of select="concat(concat('A ','N','e'),'w ','Concatenated String')"/>
 </out>
</xsl:template>
</xsl:stylesheet>