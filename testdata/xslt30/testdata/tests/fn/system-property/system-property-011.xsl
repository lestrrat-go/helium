<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="2.0">
<!-- Purpose: Test to verify that system-property($property) accepts an argument that comes from a node, variable and subsequence and the result returned is an xs:string. -->

   <xsl:variable name="version" select="//version"/>

   <xsl:variable name="vendor" select="//vendor"/>

   <xsl:variable name="ven-url" select="//ven-url"/>

   <xsl:variable name="name" select="//name"/>

   <xsl:variable name="pr-version" select="//pr-version"/>

   <xsl:variable name="is-SA" select="//is-SA"/>

   <xsl:variable name="serial" select="//serial"/>

   <xsl:variable name="bc" select="//bc"/>

   <xsl:template match="/">
      <out>
         <xsl:text>
</xsl:text>
         <version>
            <xsl:value-of select="system-property(//version) instance of xs:string"/>
            <xsl:value-of select="system-property($version) instance of xs:string"/>
            <xsl:value-of select="system-property(subsequence(('ver', 'xsl:version', //version), 3,1)) instance of xs:string"/>
         </version>
         <xsl:text>
</xsl:text>
         <vendor>
            <xsl:value-of select="system-property(//vendor) instance of xs:string"/>
            <xsl:value-of select="system-property($vendor) instance of xs:string"/>
            <xsl:value-of select="system-property(subsequence(('ver', 'xsl:vendor', //version), 2,1)) instance of xs:string"/>
         </vendor>
         <xsl:text>
</xsl:text>
         <ven-url>
            <xsl:value-of select="system-property(//ven-url) instance of xs:string"/>
            <xsl:value-of select="system-property($ven-url) instance of xs:string"/>
            <xsl:value-of select="system-property(subsequence((//ven-url, 'xsl:version', //version), 1,1)) instance of xs:string"/>
         </ven-url>
         <xsl:text>
</xsl:text>
         <name>
            <xsl:value-of select="system-property(//name) instance of xs:string"/>
            <xsl:value-of select="system-property($name) instance of xs:string"/>
            <xsl:value-of select="system-property(subsequence(('name', 'xsl:name', //name), 3,1)) instance of xs:string"/>
         </name>
         <xsl:text>
</xsl:text>
         <pr-version>
            <xsl:value-of select="system-property(//pr-version) instance of xs:string"/>
            <xsl:value-of select="system-property($pr-version) instance of xs:string"/>
            <xsl:value-of select="system-property(subsequence(('version', 'xsl:product-version', //pversion), 2,1)) instance of xs:string"/>
         </pr-version>
         <xsl:text>
</xsl:text>
         <is-SA>
            <xsl:value-of select="system-property(//is-SA) instance of xs:string"/>
            <xsl:value-of select="system-property($is-SA) instance of xs:string"/>
            <xsl:value-of select="system-property(subsequence(('is-SA', 'xsl:version', //is-SA), 3,1)) instance of xs:string"/>
         </is-SA>
         <xsl:text>
</xsl:text>
         <serial>
            <xsl:value-of select="system-property(//serial) instance of xs:string"/>
            <xsl:value-of select="system-property($serial) instance of xs:string"/>
            <xsl:value-of select="system-property(subsequence(('xsl:serial', $serial, //version), 2,1)) instance of xs:string"/>
         </serial>
         <xsl:text>
</xsl:text>
         <bc>
            <xsl:value-of select="system-property(//bc) instance of xs:string"/>
            <xsl:value-of select="system-property($bc) instance of xs:string"/>
            <xsl:value-of select="system-property(subsequence(('bc', doc, //bc), 3,1)) instance of xs:string"/>
         </bc>
         <xsl:text>
</xsl:text>
      </out>
   </xsl:template>
</xsl:transform>
