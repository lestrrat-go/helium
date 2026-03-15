<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#axes?>
    <!-- Purpose: Test for node test in argument to count() function. -->

<xsl:template match="/doc">
  <out><xsl:value-of select="count(.//comment())"/></out>
</xsl:template>

</xsl:stylesheet>
