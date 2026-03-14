<?xml version="1.0" encoding="UTF-8"?>

<!-- xsl:key test case where the match pattern uses local variables within a predicate -->

<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

	<xsl:strip-space elements="*"/>
	<xsl:key name="k"
		match="elem[some $x in dataset/field1, $y in dataset/field2 satisfies $x = $y]" use="active"/>

	<xsl:template match="/">
		<o>
			<xsl:copy-of select="key('k', 'true')"/>
		</o>
	</xsl:template>
</xsl:stylesheet>
