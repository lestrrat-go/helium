<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="2.0">
<!-- Purpose: Test case to verify that in an xsl:document any consecutive sequence of strings 
  				within the result sequence is converted to a single text node, whose string value
				contains the content of each of the strings in turn, with a single space (#x20) 
				used as a separator between successive strings.-->

   <xslt:output method="xml" encoding="UTF-8" indent="no"/>

   <xslt:template match="doc">
  		  <xslt:document>
  			    <xslt:call-template name="creator"/>
  		  </xslt:document>
	  </xslt:template>

   <xslt:template name="creator">
		    <out>
         <xslt:sequence select="'Level1','Level2','level3','level4'"/>
      </out>
	  </xslt:template>
</xslt:transform>
