<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs"
    version="3.0">
    <!-- xsl:merge test on city and weather data. -->
    <!-- the data files cities-SE.xml and weather-SE.xml were downloaded from openweathermap.org. The data
          was originally in JSON format; it was converted to XML, filtered to extract the data for Sweden,
          and sorted by city name in Swedish collating order. -->
          
    <!-- merge with a single source - another way of doing group-adjacent -->
          
    <xsl:output method="xml" indent="no"/>
    
    <xsl:template name="xsl:initial-template">
        <duplicates>
            <xsl:merge>                
                <xsl:merge-source name="cities"
                        streamable="yes"
                		for-each-source="'cities-SE.xml'"
                		select="city-list/city">
                    <xsl:merge-key select="translate(name, ' -', '')" lang="sv" case-order="upper-first"/>
                </xsl:merge-source>
                <xsl:merge-action>
                  <xsl:variable name="g" select="current-merge-group()"/>
                  <xsl:if test="$g[2]">
                    <city-name name="{current-merge-key()}">
                      <xsl:copy-of select="$g"/>
                    </city-name>  
                  </xsl:if>        
                </xsl:merge-action>
            </xsl:merge>            
        </duplicates>
    </xsl:template>
</xsl:stylesheet>