<xsl:package
    name="http://www.w3.org/xslt30tests/package-909"  
    package-version="1.0.0"
    version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:a="urn:attribute-set">
    
    <!-- tests error XTDE3052 to be raised on abstract attribute-set -->

    <xsl:attribute-set name="a:abstract" visibility="abstract">        
    </xsl:attribute-set>  

    <xsl:template name="xsl:initial-template" visibility="public">
        <not-ok xsl:use-attribute-sets="a:abstract"/>
    </xsl:template>
    
</xsl:package>   