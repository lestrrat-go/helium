<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-translate?>
  <!-- Purpose: Test of 'translate()' function with 'translate()' as an
       argument.  Only the second argument of outer 'translate' uses
       another call to 'translate' function.  -->

<xsl:template match="doc">
 <out>
   <xsl:value-of select="translate('old string',translate('123','123','old'),'new')"/>
 </out>
</xsl:template>
</xsl:stylesheet>