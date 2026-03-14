<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:fn="http://www.w3.org/2005/xpath-functions"
                xmlns:soap-env="http://schemas.xmlsoap.org/soap/envelope/" exclude-result-prefixes="fn xs soap-env">
    <xsl:output method="xml" encoding="UTF-8" indent="yes"/>

    <xsl:variable name="payload" select="/*[1]"/>

    <xsl:template match="/">
        <xsl:element name="soap-env:Envelope" namespace="http://schemas.xmlsoap.org/soap/envelope/">
            <xsl:element name="soap-env:Body">
                <xsl:element name="{fn:local-name($payload)}" namespace="{fn:namespace-uri($payload)}">
                    <xsl:copy-of select="$payload/@*"/>
                    <xsl:copy-of select="$payload/*" copy-namespaces="no"/>
                </xsl:element>
            </xsl:element>
        </xsl:element>
    </xsl:template>
    

</xsl:stylesheet>
