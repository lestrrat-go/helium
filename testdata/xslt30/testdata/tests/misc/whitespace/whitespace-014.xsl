<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test with whitespace text nodes as preceding-siblings of xsl:param. 
  				Parent has @xml:space=preserve. Whitespace must be stripped. -->

   <t:template match="doc">
      <out>
         <t:call-template name="inner">
            <t:with-param name="child" select="top-node/rubbish"/>
         </t:call-template>
         <t:apply-templates select="top-node">
            <t:with-param name="child" select="top-node/rubbish"/>
         </t:apply-templates>
      </out>
   </t:template>

   <t:template name="inner" xml:space="preserve">
   
				 
		&#xD;  
			

<t:param name="child"/>
<a><t:value-of select="$child/text()"/></a>
</t:template>

   <t:template match="top-node" xml:space="preserve">
    
				 
		&#xD;  
			

 	
<t:param name="child"/>
<b><t:value-of select="$child/text()"/></b>
</t:template>
</t:transform>
