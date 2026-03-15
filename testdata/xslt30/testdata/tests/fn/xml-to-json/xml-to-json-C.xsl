<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:j="http://www.w3.org/2005/xpath-functions"
    exclude-result-prefixes="xs j"
    version="3.0">
    
    
    <xsl:output method="xml" indent="no"/>
    
    <!-- Invalid inputs to xml-to-json -->
    
    <xsl:template name="t001">
        <xsl:variable name="in"><j:string>a</j:string><j:string>b</j:string></xsl:variable>
        <t001><xsl:sequence select="xml-to-json($in)"/></t001>
    </xsl:template>
    
    <xsl:template name="t002">
        <xsl:variable name="in"><j:date>2016-04-23</j:date></xsl:variable>
        <t002><xsl:sequence select="xml-to-json($in)"/></t002>
    </xsl:template>
    
    <xsl:template name="t003">
        <xsl:variable name="in"><j:number>NaN</j:number></xsl:variable>
        <t003><xsl:sequence select="xml-to-json($in)"/></t003>
    </xsl:template>
    
    <xsl:template name="t004">
        <xsl:variable name="in"><j:null>nothing</j:null></xsl:variable>
        <t004><xsl:sequence select="xml-to-json($in)"/></t004>
    </xsl:template>
    
    <xsl:template name="t005">
        <xsl:variable name="in"><j:number>-INF</j:number></xsl:variable>
        <t005><xsl:sequence select="xml-to-json($in)"/></t005>
    </xsl:template>
    
    <xsl:template name="t006">
        <xsl:variable name="in"><j:number>INF</j:number></xsl:variable>
        <t006><xsl:sequence select="xml-to-json($in)"/></t006>
    </xsl:template>
    
    <xsl:template name="t007">
        <xsl:variable name="in"><j:number zero="yes">0</j:number></xsl:variable>
        <t007><xsl:sequence select="xml-to-json($in)"/></t007>
    </xsl:template>
    
    <xsl:template name="t008">
        <xsl:variable name="in"><j:string>H<sub>2</sub>O</j:string></xsl:variable>
        <t008><xsl:sequence select="xml-to-json($in)"/></t008>
    </xsl:template>
    
    <xsl:template name="t009">
        <xsl:variable name="in"><j:string>H<sub>2</sub>O</j:string></xsl:variable>
        <t009><xsl:sequence select="xml-to-json($in)"/></t009>
    </xsl:template>
    
    <xsl:template name="t010">
        <xsl:variable name="in"><j:map><j:string key="k">zzzz</j:string><j:string key="k">xxxx</j:string></j:map></xsl:variable>
        <t010><xsl:sequence select="xml-to-json($in)"/></t010>
    </xsl:template>
    
    <xsl:template name="t011">
        <xsl:variable name="in"><j:array><j:null/><j:null/>abcd</j:array></xsl:variable>
        <t011><xsl:sequence select="xml-to-json($in)"/></t011>
    </xsl:template>
    
    <xsl:template name="t012">
        <xsl:variable name="in"><j:string escaped="no">xx</j:string></xsl:variable>
        <t012><xsl:sequence select="xml-to-json($in)"/></t012>
    </xsl:template>
    
    <xsl:template name="t013">
        <xsl:variable name="in"><j:map><j:string>xxxx</j:string></j:map></xsl:variable>
        <t013><xsl:sequence select="xml-to-json($in)"/></t013>
    </xsl:template>
    
    <xsl:template name="t014">
        <xsl:variable name="in"><j:map><j:string escaped-key="maybe" key="kk">xxxx</j:string></j:map></xsl:variable>
        <t014><xsl:sequence select="xml-to-json($in)"/></t014>
    </xsl:template>
    
    <xsl:template name="t015">
        <xsl:variable name="in"><j:map><j:string escaped-key="yes" key="kk">xxxx</j:string>zzzz</j:map></xsl:variable>
        <t015><xsl:sequence select="xml-to-json($in)"/></t015>
    </xsl:template>
    
    <xsl:template name="t016">
        <xsl:variable name="in"><j:string key="kk">xxxx</j:string></xsl:variable>
        <t016><xsl:sequence select="xml-to-json($in)"/></t016>
    </xsl:template>
    
    <xsl:template name="t017">
        <xsl:variable name="in"><j:string escaped="true">\Q</j:string></xsl:variable>
        <t017><xsl:sequence select="xml-to-json($in)"/></t017>
    </xsl:template>
    
    <xsl:template name="t018">
        <xsl:variable name="in"><j:string escaped="true">\uDEFG</j:string></xsl:variable>
        <t018><xsl:sequence select="xml-to-json($in)"/></t018>
    </xsl:template>
    
    <xsl:template name="t019">
        <xsl:variable name="in"><j:boolean>maybe</j:boolean></xsl:variable>
        <t019><xsl:sequence select="xml-to-json($in)"/></t019>
    </xsl:template>
    
    <xsl:template name="t020">
        <xsl:variable name="in"><j:boolean/></xsl:variable>
        <t020><xsl:sequence select="xml-to-json($in)"/></t020>
    </xsl:template>
    
    <xsl:template name="t021">
        <xsl:variable name="in"><j:boolean j:what-is-this="22">true</j:boolean></xsl:variable>
        <t021><xsl:sequence select="xml-to-json($in)"/></t021>
    </xsl:template>
    
    <xsl:template name="t022">
        <xsl:variable name="in"><j:number>\u0030</j:number></xsl:variable>
        <t022><xsl:sequence select="xml-to-json($in)"/></t022>
    </xsl:template>
    
    <xsl:template name="t023">
        <xsl:variable name="in"><j:number escaped="true">\u0030</j:number></xsl:variable>
        <t023><xsl:sequence select="xml-to-json($in)"/></t023>
    </xsl:template>
    
    <xsl:template name="t024">
        <xsl:variable name="in"><j:string escaped="true">\</j:string></xsl:variable>
        <t024><xsl:sequence select="xml-to-json($in)"/></t024>
    </xsl:template>
    
    <xsl:template name="t100">
        <xsl:variable name="in"><j:null/></xsl:variable>
        <t100><xsl:sequence select="xml-to-json($in, map{'indent': 'yes'})"/></t100>
    </xsl:template>
    
    <xsl:template name="t101">
        <xsl:variable name="in"><j:null/></xsl:variable>
        <t101><xsl:sequence select="xml-to-json($in, map{'indent': 'true'})"/></t101>
    </xsl:template>
    
    <xsl:template name="t102">
        <xsl:variable name="in"><j:null/></xsl:variable>
        <t101><xsl:sequence select="xml-to-json($in, map{'indent': ()})"/></t101>
    </xsl:template>
    
    <xsl:template name="t103">
        <xsl:variable name="in"><j:null/></xsl:variable>
        <t103><xsl:sequence select="xml-to-json($in, map{'indent': (true(), false())})"/></t103>
    </xsl:template>
    
    <xsl:template name="t104">
        <xsl:variable name="in"><j:null/></xsl:variable>
        <t103><xsl:sequence select="xml-to-json($in, map{'indent': xs:untypedAtomic('2')})"/></t103>
    </xsl:template>
    
    
    
</xsl:stylesheet>