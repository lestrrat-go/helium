<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:err="http://www.w3.org/2005/xqt-errors"
    exclude-result-prefixes="xs">

    <xsl:param name="prmXPathFile" as="xs:string" select="'try020xpath.xml'"/>
    <xsl:template match="/">
        <output>
            <xsl:for-each select="doc($prmXPathFile)//xpath">
                <xsl:variable name="xPath" select="string(.)" as="xs:string"/>
                <xsl:message select="'XPath=',$xPath"/>
                <xsl:try>
                    <xsl:evaluate xpath="string(.)"/>
                    <xsl:catch>
                        <xsl:message terminate="no"
                            select="'XPath evaluation failed: Error code:', $err:code, 'Reason:', $err:description"
                        />
                    </xsl:catch>
                </xsl:try>
            </xsl:for-each>
        </output>
    </xsl:template>
</xsl:stylesheet>
