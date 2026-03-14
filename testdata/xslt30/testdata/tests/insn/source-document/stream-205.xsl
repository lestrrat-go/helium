<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema" 
    xmlns:map="http://www.w3.org/2005/xpath-functions/map" xmlns:mf="http://example.com/mf" 
    exclude-result-prefixes="#all" version="3.0">
    
    <xsl:variable name="segregation.map" as="map(*)">
        <xsl:map>
            <xsl:map-entry key="1" select="17"/>
            <xsl:map-entry key="2" select="23"/>
        </xsl:map>
    </xsl:variable>
    
 <xsl:template name="xsl:initial-template">
        <data>
             <xsl:for-each select="map:keys($segregation.map)">
                 <xsl:variable name="grade" select="."/>
                 <xsl:for-each select="$segregation.map(.)">
                     <xsl:variable name="assignment" select="."/>
                     <xsl:source-document href="stream-205.xml" streamable="yes">
                         <xsl:apply-templates select=".//school/copy-of(.)">
                             <xsl:with-param name="grade" select="$grade"/>
                             <xsl:with-param name="assignment" select="$assignment"/>
                         </xsl:apply-templates>	
                     </xsl:source-document>
                 </xsl:for-each>
             </xsl:for-each>
         </data>          
 </xsl:template>
    
    <xsl:template match="school">
        <xsl:param name="grade"/>
        <xsl:param name="assignment"/>
        <a details="{@Genre}-{@Format}-{@Condition}" args="{$grade}-{$assignment}"/>
    </xsl:template>
</xsl:stylesheet>