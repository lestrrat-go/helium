<?xml version='1.0'?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#apply-imports?>
<!-- test that xsl:apply-imports considers template rules imported into a stylesheet
     module that includes the module containing the current template rule -->

<xsl:import href="xinc20a.xsl"/>
<xsl:include href="xinc20b.xsl"/>

<xsl:template match="root-tag">
		 <xsl:apply-templates select="one-tag"/>
</xsl:template>

</xsl:stylesheet>
