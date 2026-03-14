<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-count?>
  <!-- Purpose: Test of a function ("count") with an axes ("ancestor")
       as part of its argument. -->

<xsl:template match="doc">
 <out>
   <xsl:value-of select="count(//child1[ancestor::element1])"/>
 </out>
</xsl:template>
</xsl:stylesheet>
