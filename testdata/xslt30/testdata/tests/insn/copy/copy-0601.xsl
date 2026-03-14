<?xml version='1.0' encoding='UTF-8' ?>
<xsl:stylesheet version='2.0'
xmlns:xsl='http://www.w3.org/1999/XSL/Transform'>

<!-- test use copy-namespaces attribute of xsl:copy-of (XSLT 2.0) -->
<?spec xslt#copy-of?>   
<?same-as-1.0 no?>

<xsl:output method="xml" encoding="utf-8"/>
    <xsl:template match="/">
    	<output>
    		<with-namespaces>
    		  <xsl:copy-of select="*" copy-namespaces="yes"/>
    		</with-namespaces>
    		<without-namespaces>
    		  <xsl:copy-of select="*" copy-namespaces="no"/>
    		</without-namespaces>    		
    	</output>
    </xsl:template>
</xsl:stylesheet>
