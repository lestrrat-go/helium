<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case that uses all four instructions (xsl:apply-templates, xsl:apply-imports, 
                xsl:next-match and xsl:call-template) to test the accumulation of tunnel parameters. -->

   <t:import href="import5.xsl"/>

   <t:template match="doc">
		    <t:text>
</t:text>
		    <out>
         <t:apply-templates>
		  	       <t:with-param name="docPar" select="'doc'" tunnel="yes"/>
		       </t:apply-templates>
      </out>
	  </t:template>

   <t:template match="data">
		    <t:call-template name="A">
			      <t:with-param name="dataPar" select="'data'" tunnel="yes"/>
		    </t:call-template>
	  </t:template>

   <t:template name="A">
		    <t:apply-templates/>
	  </t:template>

   <t:template match="inner">
		    <t:next-match>
			      <t:with-param name="matchPar" select="'match'" tunnel="yes"/>
		    </t:next-match>
	  </t:template>

   <t:template match="in">
		    <t:apply-imports>
		  	    <t:with-param name="inPar" select="'in'" tunnel="yes"/>
		    </t:apply-imports>
	  </t:template>

   <t:template match="text()"/>
</t:transform>
