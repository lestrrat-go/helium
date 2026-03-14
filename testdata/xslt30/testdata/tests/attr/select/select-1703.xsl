<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0"
                exclude-result-prefixes="xs">
<!-- Purpose: Test of select in xsl:apply-templates with function call -->

   <xslt:variable name="var"> 
		    <num1 xmlns:xsl="http://www.w3.org/1999/XSL/Transform">1<num2>
            <num3/>
         </num2>
      </num1>
		    <num4 xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
         <num6/>
      </num4>
		    <num5 xmlns:xsl="http://www.w3.org/1999/XSL/Transform"/>
	  </xslt:variable>

   <xslt:variable name="seq" select="($var//*, 123.2, xs:boolean(12))"/>

   <xslt:template match="doc">
		    <out>
         <t1>
            <xslt:apply-templates select="subsequence($seq, 1, count($seq[. instance of node()]))"/>
         </t1>
      </out>
	  </xslt:template>

   <xslt:template match="@*">
		    <xslt:value-of select="name(.)"/>
      <xslt:text> | </xslt:text>
	  </xslt:template>

   <xslt:template match="*">
		    <xslt:value-of select="name(.)"/>
      <xslt:text> | </xslt:text>
	  </xslt:template>

   <xslt:template match="text()"/>
</xslt:transform>
