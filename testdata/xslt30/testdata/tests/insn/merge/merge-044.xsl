<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs"
    version="3.0">
    
    <!-- xsl:merge test on two heterogeneous files- output using the current-merge-group -->
    <!-- Also output the name of the corresponding merge source -->
    <xsl:output method="xml" indent="no"/>
    <xsl:template name="main">
        <events>
            <xsl:merge>
                <xsl:merge-source 
                	name="one"
                	select="doc('log-file-1.xml')/events/event">
                        <xsl:merge-key select="xs:dateTime(@timestamp)"/>
                </xsl:merge-source>
                <xsl:merge-source 
                	name="two"
                	select="doc('log-file-2.xml')/log/day/record">
                        <xsl:merge-key select="dateTime(../@date, time)"/>
                </xsl:merge-source>
                <xsl:merge-action>
                    <group at="{current-merge-key()}">
                        <one><xsl:copy-of select="current-merge-group('one')" /></one>
                        <two><xsl:copy-of select="current-merge-group('two')" /></two>
                    </group>
                </xsl:merge-action>
            </xsl:merge>
        </events>
        
    </xsl:template>
</xsl:stylesheet>