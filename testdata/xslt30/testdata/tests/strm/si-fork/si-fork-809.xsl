<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema" 
    xmlns:math="http://www.w3.org/2005/xpath-functions/math" 
    exclude-result-prefixes="xs math" 
    version="3.0">
    
    <xsl:param name="STREAMABLE" as="xs:boolean" static="yes" select="true()"/>
    
    <xsl:mode _streamable="{$STREAMABLE}"/>
    
    <xsl:output method="text"/>
    
    <xsl:template match="/">
        <out>
          <xsl:fork>
            <xsl:for-each-group select="Data/Entry" composite="yes" group-by="@Genre, @Condition, @Format">
                <xsl:value-of select="current-grouping-key(), sum(current-group()/@Count)" separator=", "/>
                <br/>
            </xsl:for-each-group>
          </xsl:fork>
        </out>
    </xsl:template>
    
</xsl:stylesheet>