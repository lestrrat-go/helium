<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:j="http://www.w3.org/2005/xpath-functions"
    exclude-result-prefixes="xs j"
    version="3.0">
    
    <xsl:variable name="options.liberal" as="map(*)" select="map{'liberal':true()}"/>
    <xsl:output method="xml"/>
    
    <!-- This stylesheet is designed to work either with the built-in xml-to-json function,
         or with the alternative XSLT implementation, based on the setting of a static parameter -->
    
    <xsl:param name="use-xslt-implementation" as="xs:boolean" select="false()" static="true"/>
    
    <xsl:variable name="xml-to-json-1" select="xml-to-json#1" use-when="not($use-xslt-implementation)"/>
    <xsl:variable name="xml-to-json-2" select="xml-to-json#2" use-when="not($use-xslt-implementation)"/>
    
    <xsl:use-package use-when="$use-xslt-implementation"
        name="http://www.w3.org/2013/XSLT/xml-to-json"
        package-version="1.0"/>
    
    <xsl:variable name="xml-to-json-1" 
        select="Q{http://www.w3.org/2013/XSLT/xml-to-json}xml-to-json#1" 
        use-when="$use-xslt-implementation"/>
    
    <xsl:variable name="xml-to-json-2" 
        select="Q{http://www.w3.org/2013/XSLT/xml-to-json}xml-to-json#2" 
        use-when="$use-xslt-implementation"/>
 
    
    <xsl:template name="t001">
        <t001><xsl:sequence select="$xml-to-json-1(json-to-xml('null',$options.liberal))"/></t001>
    </xsl:template>
    
    <xsl:template name="t002">
        <t002><xsl:sequence select="$xml-to-json-1(json-to-xml('true'))"/></t002>
    </xsl:template>
    
    <xsl:template name="t003">
        <t003><xsl:sequence select="$xml-to-json-1(json-to-xml('false'))"/></t003>
    </xsl:template>
    
    <xsl:template name="t004">
        <t004><xsl:sequence select="$xml-to-json-1(json-to-xml('93.7'))"/></t004>
    </xsl:template>
    
    <xsl:template name="t005">
        <xsl:variable name="in">"London"</xsl:variable>
        <t005><xsl:sequence select="$xml-to-json-1(json-to-xml($in))"/></t005>
    </xsl:template>
    
    <xsl:template name="t006">
        <xsl:variable name="in">""</xsl:variable>
        <t006><xsl:sequence select="$xml-to-json-1(json-to-xml($in))"/></t006>
    </xsl:template>
    
    <xsl:template name="t007">
        <xsl:variable name="in">[]</xsl:variable>
        <t007><xsl:sequence select="$xml-to-json-1(json-to-xml($in))"/></t007>
    </xsl:template>
    
    <xsl:template name="t008">
        <xsl:variable name="in">[34]</xsl:variable>
        <t008><xsl:sequence select="$xml-to-json-1(json-to-xml($in))"/></t008>
    </xsl:template>
    
    <xsl:template name="t009">
        <xsl:variable name="in">[true, false]</xsl:variable>
        <t009><xsl:sequence select="$xml-to-json-1(json-to-xml($in))"/></t009>
    </xsl:template>
    
    <xsl:template name="t010">
        <xsl:variable name="in">["A", "B", "C", "D", "E"]</xsl:variable>
        <t010><xsl:sequence select="$xml-to-json-1(json-to-xml($in))"/></t010>
    </xsl:template>
    
    <xsl:template name="t011">
        <xsl:variable name="in">{}</xsl:variable>
        <t011><xsl:sequence select="$xml-to-json-1(json-to-xml($in))"/></t011>
    </xsl:template>
    
    <xsl:template name="t012">
        <xsl:variable name="in">{"A":true}</xsl:variable>
        <t012><xsl:sequence select="$xml-to-json-1(json-to-xml($in))"/></t012>
    </xsl:template>
    
    <xsl:template name="t013">
        <xsl:variable name="in">{"A":1, "B":2, "C":3, "D":4, "E":5}</xsl:variable>
        <t013><xsl:sequence select="$xml-to-json-1(json-to-xml($in))"/></t013>
    </xsl:template>
    
    <xsl:template name="t014">
        <xsl:variable name="in">{"A":[1,2], "B":[true,false], "C":["Paris","London","Berlin"], "D":[1, true, "on"], "E":[null, null]}</xsl:variable>
        <t014><xsl:sequence select="$xml-to-json-1(json-to-xml($in))"/></t014>
    </xsl:template>
    
    <xsl:template name="t015">
        <xsl:variable name="in">[[], 1]</xsl:variable>
        <t015><xsl:sequence select="$xml-to-json-1(json-to-xml($in))"/></t015>
    </xsl:template>
    
    <xsl:template name="t016">
        <xsl:variable name="in">[{}, 1]</xsl:variable>
        <t016><xsl:sequence select="$xml-to-json-1(json-to-xml($in))"/></t016>
    </xsl:template>
    
    <xsl:template name="t017">
        <xsl:variable name="in">{"A":{}, "B":1}</xsl:variable>
        <t017><xsl:sequence select="$xml-to-json-1(json-to-xml($in))"/></t017>
    </xsl:template>
    
    <xsl:template name="t018">
        <xsl:variable name="in">{"A":[], "B":1}</xsl:variable>
        <t018><xsl:sequence select="$xml-to-json-1(json-to-xml($in))"/></t018>
    </xsl:template>
    
    <xsl:template name="t019">
        <xsl:variable name="in"><j:map><j:string key='\"' escaped-key=" true ">1</j:string><j:string key='\"' escaped-key=" false ">0</j:string></j:map></xsl:variable>
        <t><xsl:sequence select="$xml-to-json-1($in)"/></t>
    </xsl:template>
    
  
    <xsl:template name="e001">
        <xsl:variable name="in">"\\\/\"\r\t\n\u0020"</xsl:variable>
        <e001><xsl:sequence select="$xml-to-json-1(json-to-xml($in)/*)"/></e001>
    </xsl:template>
    
    <xsl:template name="f001">
        <xsl:variable name="in" as="element()">
            <a><b/></a>
        </xsl:variable>
        <xsl:variable name="output" as="element()">
            <output:serialization-parameters xmlns:output="http://www.w3.org/2010/xslt-xquery-serialization">
                <output:cdata-section-elements value="f001"/>
                <output:omit-xml-declaration value="yes"/>
            </output:serialization-parameters>
        </xsl:variable>
        <f001><xsl:sequence select="$xml-to-json-2($in, map{ 'fallback' : serialize(?, $output) })"/></f001>
    </xsl:template>
    
    
    
</xsl:stylesheet>