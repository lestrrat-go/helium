<?xml version='1.0' encoding='UTF-8' ?>
<xsl:stylesheet version='3.0'
xmlns:xsl='http://www.w3.org/1999/XSL/Transform'>

	<!-- Test case that uses copy-namespaces ="TRUE" on an xsl:copy-of instruction (not
         allowed) -->
<?spec xslt#copy-of?>   
<?same-as-1.0 no?>

<xsl:output method="xml" encoding="utf-8"/>
    <xsl:template match="/">
    	<output>
    		<with-namespaces>
    		  <xsl:copy-of select="*" copy-namespaces="TRUE"/>
    		</with-namespaces>
    		<without-namespaces>
    		  <xsl:copy-of select="*" copy-namespaces="0"/>
    		</without-namespaces>    		
    	</output>
    </xsl:template>
</xsl:stylesheet>
