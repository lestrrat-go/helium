<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
				xmlns:ped="http://tester.com"
				xmlns:ljh="http://buster.com"
				xmlns:jad="http://administrator.com">

<?spec xslt#lre-namespaces?>
    <!-- Purpose: xsl:exclude-result-prefixes should 
       only work to omit namespace declarations that are not actually used. -->

<xsl:template match="doc">
  <out x="by the corner"  xsl:exclude-result-prefixes="ped">
    <jad:output1/>
	<jad:output2>
		<jad:output2a/>
	</jad:output2>
	<ljh:output1/>
	<ljh:output2>
		<ljh:output2a/>
	</ljh:output2>
  </out>
</xsl:template>

</xsl:stylesheet>
