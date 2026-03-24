	<xsl:stylesheet
	 version="2.0"
	 xmlns:xsl='http://www.w3.org/1999/XSL/Transform'
	>

    <?spec xslt#variables?>
	<!-- BUG: Saxon 5.3.2/016 -->
    <!-- DESCRIPTION: A null pointer exception occurs if a local variable is declared within
    the template-body of a global variable declaration. -->

	<xsl:variable name="first-two">
	  <xsl:variable name="paras" select="//p" />
	  <xsl:value-of select="concat($paras[1], '+', $paras[2])" />
	</xsl:variable>

	<xsl:param name="next-two">
	  <xsl:variable name="paras" select="//p" />
	  <xsl:value-of select="concat($paras[2], '+', $paras[3])" />
	</xsl:param>

	<xsl:template match="/">
	<out>
	<xsl:value-of select="$first-two"/>;
	<xsl:value-of select="$next-two"/>
	</out>
	</xsl:template>

	</xsl:stylesheet>

