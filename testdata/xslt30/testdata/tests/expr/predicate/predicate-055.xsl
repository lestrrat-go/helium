<?xml version="1.0" encoding="UTF-8"?>

<xsl:transform version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xhtml="http://www.w3.org/1999/xhtml">
    
    <xsl:output method="xml" indent="no" encoding="UTF-8"/>
    <xsl:strip-space elements="*"/>
    <xsl:preserve-space elements="xhtml:span"/>
    
    <xsl:template match="/">
        <out>
            <xsl:if test="doc-available(concat(xhtml:html/@xml:id,'.html')) and doc(concat(xhtml:html/@xml:id,'.html'))/xhtml:html/xhtml:body/xhtml:div[@class = 'ocr_page']"> <!-- fails -->
                <xsl:text>Correct</xsl:text>
            </xsl:if>
        </out>
    </xsl:template>
    
</xsl:transform>