<?xml version="1.0" ?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<xsl:import href="import2.xsl" />

	<xsl:template match="doc">
		<xsl:apply-templates>
			<xsl:with-param name="par" select="'Toronto'" tunnel="yes"/>
			<xsl:with-param name="par2" select="'Montreal'" tunnel="no"/>
			<xsl:with-param name="par3" select="'halifax'" />
		</xsl:apply-templates>
	</xsl:template>	
  
</xsl:stylesheet>