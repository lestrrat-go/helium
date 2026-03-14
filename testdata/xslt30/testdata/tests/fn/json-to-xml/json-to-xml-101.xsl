<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:j="http://www.w3.org/2005/xpath-functions"
    exclude-result-prefixes="xs j" expand-text="yes" version="3.0">

    <xsl:variable name="input" expand-text="no"> {"d": {"results": [ { "TYPE_ID": "20",
        "QUESTION_ID": "41333", "RESPONSE_TYPE": "Checkboxes", "QUESTIONVAL": "xssdsdd", "RESPONSE":
        "Yes", "RESPONSE_ORDER": "1" }, { "TYPE_ID": "21", "QUESTION_ID": "417", "RESPONSE_TYPE":
        "Radio buttons", "QUESTIONVAL": "abc", "RESPONSE": "Yes,No", "RESPONSE_ORDER": "1,2" } ]}} </xsl:variable>


    <xsl:mode on-no-match="shallow-copy"/>

    <xsl:template name="main">
        <xsl:variable name="inputAsXml" select="json-to-xml($input)"/>
        <xsl:variable name="outputAsXml">
            <xsl:apply-templates select="$inputAsXml"/>
        </xsl:variable>
        <out><xsl:value-of select="xml-to-json($outputAsXml)"/></out>
    </xsl:template>

    <xsl:template match="*[@key='d']">
        <xsl:apply-templates/>
    </xsl:template>

    <xsl:template match="*[@key='RESPONSE_ORDER']"/>

    <xsl:template match="*[@key='RESPONSE']">
        <xsl:variable name="response" select="tokenize(., ',')"/>
        <xsl:variable name="responseOrder" select="tokenize(../*[@key='RESPONSE_ORDER'], ',')"/>
        <j:array key="Response&amp;Order">
            <xsl:for-each select="1 to count($response)">
                <xsl:variable name="i" select="."/>
                <j:map>
                    <j:string key="RESPONSE">{$response[$i]}</j:string>
                    <j:string key="RESPONSE_ORDER">{$responseOrder[$i]}</j:string>
                </j:map>
            </xsl:for-each>
        </j:array>
    </xsl:template>

</xsl:stylesheet>
