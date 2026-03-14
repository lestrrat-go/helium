<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:local="http://local/"
    exclude-result-prefixes="xs"
    version="3.0">
    
    <!-- inline function literal, constructor function, user-defined atomic type  -->
    
    <xsl:param name="b" select="true()"/>    

    <xsl:import-schema>
        <xs:schema targetNamespace="http://local/">
            <xs:simpleType name="abc123">
                <xs:restriction base="xs:string">
                    <xs:pattern value="[a-z][a-z][a-z][0-9][0-9][0-9]"/>
                </xs:restriction>
            </xs:simpleType>
        </xs:schema>
    </xsl:import-schema>
    
    <xsl:template name="main">
        <out><xsl:value-of select="let $f := (if ($b) then local:abc123#1 else string#1) return $f('xyz987')"/></out>
    </xsl:template>
    
</xsl:stylesheet>