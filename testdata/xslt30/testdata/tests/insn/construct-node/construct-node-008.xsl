<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case to verify that any consecutive sequence of strings within the 
  				result sequence is converted to a single text node, whose string value 
  				contains the content of each of the strings in turn, with a single space 
  				(#x20) used as a separator between successive strings.
  				Test case uses a sequence of literal text in the select attribute of an 
  				xsl:value-of instruction with a deafult separator.-->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:variable name="v">
		    <t:sequence select="'Level1','Level2','level3','level4'"/>
	  </t:variable>

   <t:template match="doc">
  		  <out>
         <t:call-template name="creator"/>
      </out>
	  </t:template>

   <t:template name="creator">
  		  <t:value-of select="$v"/>
	  </t:template>
</t:transform>
