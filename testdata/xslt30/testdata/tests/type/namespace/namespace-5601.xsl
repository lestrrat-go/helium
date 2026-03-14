<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
                xmlns:tst="http://test/"
                exclude-result-prefixes="tst">

<?spec fo#func-local-name?>
    <!-- Purpose: Test of local-name() on default namespace declaration. -->

<xsl:template match="/tst:a">
  <out>
   	<xsl:value-of select="local-name(namespace::*[string()='http://www.w3.org/1999/XMLSchema-instance'])"/>,
<xsl:value-of select="local-name(namespace::*[string()='http://test/'])"/>
  </out>
</xsl:template>

</xsl:stylesheet>