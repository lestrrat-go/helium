<xsl:transform 
  version="2.0" 
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema">
  
  <!-- Demonstrate error when using group-starting-with when population 
       contains atomic values -->
  <!-- No longer an error with XSLT 3.0 -->     
       <?spec xslt#grouping?>
       <?error XTTE1120?>

<xsl:output indent="yes"/>
  
<xsl:template match="/">
   <out>
    <xsl:for-each-group select="1,2,3,4,2,1" group-starting-with="/">
      <group>
        <xsl:value-of select="current-group()"/>
      </group>
    </xsl:for-each-group>
   </out>
</xsl:template>    
	
</xsl:transform>