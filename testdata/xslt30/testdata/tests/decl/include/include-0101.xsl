<?xml version='1.0'?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<xsl:import href="include-0101a.xsl"/>
<xsl:include href="include-0101b.xsl"/>


<xsl:template match="root-tag">
		 <body changed="8 Feb 2002" var="{$var}">
		 <xsl:apply-templates/><br/>
		 </body>
</xsl:template>

</xsl:stylesheet>
