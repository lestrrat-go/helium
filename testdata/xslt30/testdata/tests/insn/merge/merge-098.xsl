<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="#all">
    
    <xsl:mode streamable="yes"/>
    
    <xsl:output indent="yes"/>
    
    <xsl:template match="/">
        <AllEmployeeData>
            <xsl:variable name="input-uri" as="xs:anyURI" select="base-uri()"/>
            <!--<xsl:message select="$input-uri"/>-->
            <xsl:merge>
                <xsl:merge-source name="emp-details1" for-each-source="$input-uri" streamable="yes" select="Root/EmployeePersonal/Employee">
                    <xsl:merge-key select="ID"/>
                </xsl:merge-source>
                <xsl:merge-source name="emp-details2" for-each-source="$input-uri" streamable="yes" select="Root/EmployeeEmployment/Empl">
                    <xsl:merge-key select="Emplid"/>
                </xsl:merge-source>
                <xsl:merge-action>
                    <xsl:copy>
                        <xsl:copy-of select="current-merge-group('emp-details1')/*, current-merge-group('emp-details2')/(* except Emplid)"/>
                    </xsl:copy>
                </xsl:merge-action>
            </xsl:merge>
        </AllEmployeeData>
    </xsl:template>
    
</xsl:stylesheet>