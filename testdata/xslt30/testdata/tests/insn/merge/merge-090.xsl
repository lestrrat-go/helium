<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs" version="3.0">
    
    <xsl:output indent="yes" />
    <xsl:strip-space elements="*" />
    <xsl:mode on-no-match="shallow-copy" />
    
    <xsl:template match="/" name="main">
        <Personenliste>
            <xsl:merge>
                <xsl:merge-source for-each-source="'merge-090.xml'" streamable="yes" select="Company/Person/Name">
                    <xsl:merge-key select="Surname" />
                </xsl:merge-source>
                <xsl:merge-action>
                    <xsl:apply-templates select="current-merge-group()" />
                </xsl:merge-action>
            </xsl:merge>
        </Personenliste>
    </xsl:template>
    
    <xsl:template match="Name">
        <Person>
            <xsl:attribute name="Firma" select="ancestor::Company/@Name"></xsl:attribute>
            <xsl:apply-templates select="node()" />
        </Person>
    </xsl:template>
    
    <xsl:template match="Surname">
        <Nachname>
            <xsl:apply-templates select="node()" />
        </Nachname>
    </xsl:template>
    
    <xsl:template match="Firstname">
        <Vorname>
            <xsl:apply-templates select="node()" />
        </Vorname>
    </xsl:template>
    
</xsl:stylesheet>