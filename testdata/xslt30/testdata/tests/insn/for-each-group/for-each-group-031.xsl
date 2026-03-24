<?spec xslt#grouping?>
<xsl:transform 
  version="2.0" 
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs">


<!-- A contrived test to ensure that variables - specifically, a variable
     that's subject to optimization - can be handled in a for-each-group pattern -->
     
<xsl:param name="param" select="false()" as="xs:boolean"/>
     
<xsl:template match="body">
<chapter>
    <xsl:for-each select="*/..">
    <xsl:variable name="p" select="if ($param) then 'yes' else 'no'"/>
	<xsl:for-each-group select="*" group-ending-with="*[@cont=$p]"	>
	    <section>
	       <xsl:for-each select="current-group()">
	           <para><xsl:value-of select="."/></para>
	       </xsl:for-each> 
	   </section>
	</xsl:for-each-group>
	</xsl:for-each>
</chapter>
</xsl:template>
	
</xsl:transform>