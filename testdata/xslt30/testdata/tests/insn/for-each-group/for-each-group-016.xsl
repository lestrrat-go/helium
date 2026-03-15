<?spec xslt#grouping?>
<xsl:transform 
  version="2.0" 
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  
  <!-- simple test of positional grouping using group-starting-with -->
  
<xsl:template match="body">
<chapter>
	<xsl:for-each-group select="*" group-starting-with="h2"	>
	    <section title="{self::h2}">
	       <xsl:for-each select="current-group()[self::p]">
	           <para><xsl:value-of select="."/></para>
	       </xsl:for-each> 
	   </section>
	</xsl:for-each-group>
</chapter>
</xsl:template>
	
</xsl:transform>