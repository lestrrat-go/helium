<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
  xmlns:xs="http://www.w3.org/2001/XMLSchema" 
  exclude-result-prefixes="xs" 
  version="2.0">
  
   
  <xsl:template match="/"> 
    <out>
      <xsl:variable name="duplicates" as="element()*">      
        <xsl:for-each-group select="/root/items/item" group-by="@key">
          <xsl:sequence select="current-group()[current-group()[2]]"/>
        </xsl:for-each-group>
      </xsl:variable>
    <xsl:apply-templates select="$duplicates"/>
    </out>
  </xsl:template>
  
  <xsl:template match="item">
    <item value="{@value}" pos="{position()}" last="{last()}"/>
    <!--<xsl:message select="'position(): ', position(), '; last(): ', last()"/>
    {
    
    "value" : "<xsl:value-of select="@value"/>" 
    
    }<xsl:if test="position() &lt; last()">,</xsl:if>-->
  </xsl:template>
  
</xsl:stylesheet>