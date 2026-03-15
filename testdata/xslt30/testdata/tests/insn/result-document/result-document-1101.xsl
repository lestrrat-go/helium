<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Call to xsl:result-document is not allowed in temporary output state -->

   <xsl:template name="a">
	  <xsl:variable name="v">
	     <xsl:call-template name="rrr"/>
	  </xsl:variable>
	  <xsl:sequence select="$v"/>
   </xsl:template>
   
   <xsl:template name="b">
     <b>
	  <xsl:attribute name="v">
	     <xsl:call-template name="rrr"/>
	  </xsl:attribute>
	 </b>
   </xsl:template>
   
   <xsl:template name="c">
     <c>
	  <xsl:value-of>
	     <xsl:call-template name="rrr"/>
	  </xsl:value-of>
	 </c>
   </xsl:template>
   
   <xsl:template name="d">
     <d>
	  <xsl:comment>
	     <xsl:call-template name="rrr"/>
	  </xsl:comment>
	 </d>
   </xsl:template>
   
   <xsl:template name="e">
     <e>
	  <xsl:processing-instruction name="pi">
	     <xsl:call-template name="rrr"/>
	  </xsl:processing-instruction>
	 </e>
   </xsl:template>
   
   <xsl:template name="f">
     <f>
	  <xsl:namespace name="n">
	     <xsl:call-template name="rrr"/>
	  </xsl:namespace>
	 </f>
   </xsl:template>
   
   <xsl:template name="g">
     <g>
      <xsl:perform-sort select="1 to 100">      
	    <xsl:sort>
	     <xsl:call-template name="uuu"/>
	    </xsl:sort>
	  </xsl:perform-sort>  
	 </g>
   </xsl:template>
   
   <xsl:template name="h">
     <h>
	  <xsl:message>
	     <xsl:call-template name="rrr"/>
	  </xsl:message>
	 </h>
   </xsl:template>
   
   <xsl:template name="i">
     <i>
	  <xsl:call-template name="sss">
	     <xsl:with-param name="n">
	        <xsl:call-template name="rrr"/>
	     </xsl:with-param>   
	  </xsl:call-template>
	 </i>
   </xsl:template>
   
   <xsl:template name="j">
     <j>
	  <xsl:call-template name="ttt"/>
	 </j>
   </xsl:template>
   
   <xsl:key name="k" match="*">
     <xsl:call-template name="rrr"/>
   </xsl:key>  
   
   <xsl:template name="k">
     <xsl:variable name="v">
       <a>1</a>
       <b>2</b>
       <c>3</c>
     </xsl:variable>
     <k>
	  <xsl:sequence select="key('k', '2', $v)"/>
	 </k>
   </xsl:template>
   
   
   
   
   
   <xsl:template name="rrr">
     <xsl:result-document href="out.xml"><boo/></xsl:result-document>
     <xsl:text>rrr</xsl:text>
   </xsl:template>
   
   <xsl:template name="sss">
     <xsl:param name="n"/>
     <foo n="{$n}"/>
   </xsl:template>
   
   <xsl:template name="ttt">
     <xsl:param name="n">
       <xsl:call-template name="rrr"/>
     </xsl:param>  
     <foo n="{$n}"/>
   </xsl:template>
   
   <xsl:template name="uuu">
     <xsl:result-document href="out{.}.xml"><boo/></xsl:result-document>
     <xsl:sequence select="."/>
   </xsl:template>
   
</xsl:transform>
