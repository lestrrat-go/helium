<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema" 
    version="3.0">
    
    <xsl:param name="extract-products" as="xs:string*" select="'product-1', 'product-3'"/>
    
    <xsl:param name="input-uri" as="xs:string" select="'stream-210.xml'"/>
    
    <xsl:template name="xsl:initial-template">
        <xsl:iterate select="$extract-products">
            <xsl:variable name="product-type" as="xs:string" select="."/>
            <xsl:result-document href="{$product-type}-result.xml">
                <root>
                    <xsl:source-document streamable="true" href="{$input-uri}">
                        <xsl:for-each select="*/product[@type = $product-type]">
                            <xsl:copy-of select="."/>
                        </xsl:for-each>
                    </xsl:source-document>
                </root>
            </xsl:result-document>
        </xsl:iterate>
    </xsl:template>
    
</xsl:stylesheet>