<?xml version="1.0" encoding="UTF-8"?>

<!-- Test case submitted on SourceForge saxon-help forum on 2008-01-23 by joelfisler.
     Tests use of a local variable in the count pattern  -->

<xsl:stylesheet version="2.0" 
xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
	<xsl:param name="role" select="'tutor'"/>
	<xsl:template match="/lesson | unit">
	<out>
		<xsl:number level="multiple" count="lesson | unit[@role=$role]" format="(1) "/>
		<xsl:apply-templates/>
	</out>	
	</xsl:template>
</xsl:stylesheet>