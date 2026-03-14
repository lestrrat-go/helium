<?xml version="1.0" encoding="utf-8"?>
	<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
	                version="2.0">
	                
	  <!-- Type error: applying templates to a variable of type string -->
      <?spec xslt#applying-templates?>
      <?error XTTE0520?>
	
	  <xsl:template match="/">
	
	    <xsl:variable
	      name="empty"/>

	
	    <xsl:apply-templates select="$empty"/>
	
	  </xsl:template>
	
	</xsl:stylesheet>
