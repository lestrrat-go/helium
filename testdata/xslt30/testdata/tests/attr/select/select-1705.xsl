<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0"
                exclude-result-prefixes="xs">
<!-- Purpose: Test of select in xsl:value-of with complex pathExpr involving various node and name tests -->

   <xslt:variable name="var">
		    <doc xmlns:xsl="http://www.w3.org/1999/XSL/Transform">at1at2at3<num1>1<num2>2<num3/>
            </num2>
         </num1>
         <num4>4<num6>6</num6>
         </num4>
         <num5>5</num5>
      </doc>
	  </xslt:variable>

   <xslt:template match="doc">
		    <out>
         <t1>
            <xslt:choose>
    				       <xslt:when test="name(../child::element()/attribute()[string(.) ne 'at2'][1]) = 'att1'">
      					       <xslt:value-of select="../child::element()/attribute()[string(.) ne 'at2']"
                                 separator="|"/>
    				       </xslt:when>
   					       <xslt:otherwise>
      					       <xslt:value-of select="reverse(../child::element()/attribute()[string(.) ne 'at2'])"
                                 separator="|"/>
    				       </xslt:otherwise>
  				      </xslt:choose>
         </t1>
         <t2>
            <xslt:value-of select="($var/element()//child::*)[*]" separator="|"/>
         </t2>
      </out>
	  </xslt:template>

   <xslt:template match="text()"/>
</xslt:transform>
