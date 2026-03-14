<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xmlns:xs="http://www.w3.org/2001/XMLSchema"
    >
    
    <xsl:template match="/" name="xsl:initial-template">
        <data>
            <xsl:for-each select="1">
                <xsl:variable name="var" select="."/>
                <xsl:source-document href="{'stream-205.xml'}" streamable="yes">
                    <xsl:apply-templates select="/Data/school/copy-of(.)">
                        <xsl:with-param name="assignment" select="$var"/>
                    </xsl:apply-templates>	
                </xsl:source-document>
            </xsl:for-each>
        </data>     
    </xsl:template>
    
    <xsl:template match="school">
        <xsl:param name="assignment"/>
        <a details="{@Genre}-{@Format}-{@Condition}" args="{$assignment}"/>
    </xsl:template>
    
</xsl:stylesheet>