<?xml version="1.0" encoding="UTF-8"?>
<!-- test case for a bug found in Saxon 6.5.2 -->

<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xmlns:d="http://mydata.com/">
    
    <d:Footnote indicator="$#001">
        <Instance>
            <Codes>
                <Code>709011</Code>
            </Codes>
            <Footnote>ABC</Footnote>
        </Instance>
        <Instance>
            <Footnote>DEF</Footnote>
        </Instance>
    </d:Footnote>
    <xsl:variable name="data" select="document('')/*/d:Footnote"/>
    <xsl:template match="text()"/>
    <xsl:key name="courseCodes" match="Code" use="generate-id(ancestor::Course)"/>
    <!--
    Adjust the contents to footnote atributes on the basis of the conversion Data
    -->
    <xsl:template match="Quota">
        <xsl:variable name="footnoteCode" select="$data[@indicator = current()/@footnote][1]/Instance[(key('courseCodes', generate-id(ancestor::Course)) = Codes/Code) or not(Codes)]/Footnote[1]"/>
        <Output>
            <FootnoteCodes>
                <xsl:copy-of select="$footnoteCode" copy-namespaces="no"/>
            </FootnoteCodes>
        </Output>
    </xsl:template>
</xsl:stylesheet>
