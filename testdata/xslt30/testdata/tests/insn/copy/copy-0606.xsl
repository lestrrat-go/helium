<?xml version='1.0' encoding='UTF-8' ?>
<xsl:stylesheet version='3.0'
xmlns:xsl='http://www.w3.org/1999/XSL/Transform'>

    <!-- test use copy-namespaces attribute of xsl:copy, with values "  1"  (with some white-space), "0". (XSLT 3.0) -->
<?spec xslt#shallow-copy?>   
<?same-as-1.0 no?>

<xsl:output method="xml" encoding="utf-8"/>
    <xsl:template match="/">
    	<output>
    		<with-namespaces>
    		  <xsl:apply-templates mode="copy"/>
    		</with-namespaces>
    		<without-namespaces>
    		  <xsl:apply-templates mode="no-copy"/>
    		</without-namespaces>    		
    	</output>
    </xsl:template>
    
    <xsl:template match="*" mode="copy">
        <xsl:copy copy-namespaces="  1">
        <xsl:copy-of select="@*"/>
        <xsl:apply-templates mode="#current"/>
        </xsl:copy>
    </xsl:template>
    
    <xsl:template match="*" mode="no-copy">
        <xsl:copy copy-namespaces="0">
        <xsl:copy-of select="@*"/>
        <xsl:apply-templates mode="#current"/>
        </xsl:copy>
    </xsl:template>    
    
</xsl:stylesheet>
