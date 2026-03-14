<?xml version="1.0" encoding="utf-8"?>

   <!-- tests a forwards reference to a function within a variable declaration -->
	
	<?spec xslt#stylesheet-functions?>
	<xsl:stylesheet
	  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
	
	  xmlns:ns="http://namespace/"
	
	  version="2.0">
	
	  <xsl:template match="/">
	    <xsl:variable 
	      name="variable" 
	      select="ns:function()"/>
	    <z><xsl:copy-of select="$variable"/></z>
	  </xsl:template>
	
	  <xsl:function name="ns:function">
	    <xsl:sequence
	      select="'gogo'"/>
	  </xsl:function>
	
	</xsl:stylesheet>
