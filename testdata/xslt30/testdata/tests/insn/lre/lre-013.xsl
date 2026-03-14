<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
				xmlns:ped="http://tester.com"
				xmlns:ljh="http://buster.com"
				xmlns:jad="http://administrator.com">

    <!-- Purpose: xsl:exclude-result-prefixes should 
       only work to omit namespace declarations that are not actually used. -->

<xsl:template match="doc">
  <sits x="little jack horner" xsl:exclude-result-prefixes="ped jad"/>
</xsl:template>

</xsl:stylesheet>
