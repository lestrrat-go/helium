<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs" version="2.0">
    
    <!-- Test effect of validation on ID and IDREFS attributes -->
    
    <xsl:output method="xml" indent="no"/>

    <xsl:import-schema>
        <xs:schema>
            <xs:element name="data">
                <xs:complexType>
                    <xs:sequence>
                        <xs:element ref="thing" minOccurs="0" maxOccurs="unbounded"/>
                    </xs:sequence>
                </xs:complexType>
            </xs:element>
            <xs:element name="thing">
                <xs:complexType>
                    <xs:attribute name="id" type="xs:ID"/>
                    <xs:attribute name="parts" type="xs:IDREFS"/>
                </xs:complexType>
            </xs:element>
        </xs:schema>
    </xsl:import-schema>
    
    <xsl:variable name="data">
        <data>
            <thing id="p1" parts="q1 q2 q3"/>
            <thing id="q1" parts="r1 r2"/>
            <thing id="q2"/>
            <thing id="q3"/>
            <thing id="r1"/>
            <thing id="r2"/>
        </data>
    </xsl:variable>
    
    <xsl:variable name="bad-data-1">
        <data>
            <thing id="p1" parts="q1 q2 q3"/>
            <thing id="q1" parts="r1 r2"/>
            <thing id="q2"/>
            <thing id="q3"/>
            <thing id="r1"/>
            <thing id="r2"/>
            <thing id="q2"/> <!-- duplicate -->
        </data>
    </xsl:variable>
    
    <xsl:variable name="bad-data-2">
        <data>
            <thing id="p1" parts="q1 q2 q3"/>
            <thing id="q1" parts="r1 r2"/>
            <thing id="q2" parts="s1"/> <!-- dangling -->
            <thing id="q3"/>
            <thing id="r1"/>
            <thing id="r2"/>
        </data>
    </xsl:variable>
    
    <xsl:template name="c-001">
        <xsl:copy-of select="$data" validation="strict"/>
    </xsl:template>
    
    <xsl:template name="c-002">
        <xsl:copy-of select="$data" validation="lax"/>
    </xsl:template>
    
    <xsl:template name="c-003">
        <xsl:copy-of select="$data" validation="preserve"/>
    </xsl:template>
    
    <xsl:template name="c-004">
        <xsl:copy-of select="$data" validation="strip"/>
    </xsl:template>
    
    <xsl:template name="c-011">
        <xsl:copy-of select="$bad-data-1" validation="strict"/>
    </xsl:template>
    
    <xsl:template name="c-012">
        <xsl:copy-of select="$bad-data-1" validation="lax"/>
    </xsl:template>
    
    <xsl:template name="c-013">
        <xsl:copy-of select="$bad-data-1" validation="preserve"/>
    </xsl:template>
    
    <xsl:template name="c-014">
        <xsl:copy-of select="$bad-data-1" validation="strip"/>
    </xsl:template>
    
    <xsl:template name="c-021">
        <xsl:copy-of select="$bad-data-2" validation="strict"/>
    </xsl:template>
    
    <xsl:template name="c-022">
        <xsl:copy-of select="$bad-data-2" validation="lax"/>
    </xsl:template>
    
    <xsl:template name="c-023">
        <xsl:copy-of select="$bad-data-2" validation="preserve"/>
    </xsl:template>
    
    <xsl:template name="c-024">
        <xsl:copy-of select="$bad-data-2" validation="strip"/>
    </xsl:template>
    
    <xsl:variable name="valid-data">
        <xsl:copy-of select="$data" validation="strict"/>
    </xsl:variable>
    
    <xsl:template name="c-031">
        <xsl:variable name="v">
            <xsl:copy-of select="$valid-data" validation="strict"/>
        </xsl:variable>
        <out>
            <a><xsl:value-of select="count(id('p1', $v))"/></a>
            <b><xsl:value-of select="count(idref('r1', $v))"/></b>
        </out>
    </xsl:template>
    
    <xsl:template name="c-032">
        <xsl:variable name="v">
            <xsl:copy-of select="$valid-data" validation="lax"/>
        </xsl:variable>
        <out>
            <a><xsl:value-of select="count(id('p1', $v))"/></a>
            <b><xsl:value-of select="count(idref('r1', $v))"/></b>
        </out>
    </xsl:template>
    
    <xsl:template name="c-033">
        <xsl:variable name="v">
            <xsl:copy-of select="$valid-data" validation="preserve"/>
        </xsl:variable>
        <out>
            <a><xsl:value-of select="count(id('p1', $v))"/></a>
            <b><xsl:value-of select="count(idref('r1', $v))"/></b>
        </out>
    </xsl:template>
    
    <xsl:template name="c-034">
        <xsl:variable name="v">
            <Z><xsl:copy-of select="$valid-data" validation="strip"/></Z>
        </xsl:variable>
        <out>
            <a><xsl:value-of select="count(id('p1', $v))"/></a>
            <b><xsl:value-of select="count(idref('r1', $v))"/></b>
        </out>
    </xsl:template>
    

</xsl:stylesheet>
