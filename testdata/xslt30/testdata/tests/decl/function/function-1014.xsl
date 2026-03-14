<?spec xslt#stylesheet-functions?>
<xsl:stylesheet version="2.0"
xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
xmlns:xs="http://www.w3.org/2001/XMLSchema"
xmlns:f="http://fxsl.sf.net/"
xmlns:maximum-pick-bigger="f:maximum-pick-bigger"
xmlns:Mycompare="http://Mycompare"
exclude-result-prefixes="f xs maximum-pick-bigger Mycompare"
>

<!-- One of many tests from Dimitre Novatchev's FXSL library.
     Tests FXSL max() function.
     Exercises recursion including tail call optimization -->

  <xsl:function name="f:apply">
   <xsl:param name="pFunc" as="element()"/>
   <xsl:param name="arg1"/>
   
    <xsl:apply-templates select="$pFunc" mode="f:FXSL">
      <xsl:with-param name="arg1" select="$arg1"/>
    </xsl:apply-templates>
  </xsl:function>

  <xsl:function name="f:apply">
   <xsl:param name="pFunc" as="element()"/>
   <xsl:param name="arg1"/>
   <xsl:param name="arg2"/>
   
    <xsl:apply-templates select="$pFunc" mode="f:FXSL">
      <xsl:with-param name="arg1" select="$arg1"/>
      <xsl:with-param name="arg2" select="$arg2"/>
    </xsl:apply-templates>
  </xsl:function>


  <xsl:function name="f:apply">
   <xsl:param name="pFunc" as="element()"/>
   <xsl:param name="arg1"/>
   <xsl:param name="arg2"/>
   <xsl:param name="arg3"/>
   
    <xsl:apply-templates select="$pFunc" mode="f:FXSL">
      <xsl:with-param name="arg1" select="$arg1"/>
      <xsl:with-param name="arg2" select="$arg2"/>
      <xsl:with-param name="arg3" select="$arg3"/>
    </xsl:apply-templates>
  </xsl:function>

  <xsl:function name="f:apply">
   <xsl:param name="pFunc" as="element()"/>
   <xsl:param name="arg1"/>
   <xsl:param name="arg2"/>
   <xsl:param name="arg3"/>
   <xsl:param name="arg4"/>
   
    <xsl:apply-templates select="$pFunc" mode="f:FXSL">
      <xsl:with-param name="arg1" select="$arg1"/>
      <xsl:with-param name="arg2" select="$arg2"/>
      <xsl:with-param name="arg3" select="$arg3"/>
      <xsl:with-param name="arg4" select="$arg4"/>
    </xsl:apply-templates>
  </xsl:function>

  <xsl:function name="f:apply">
   <xsl:param name="pFunc" as="element()"/>
   <xsl:param name="arg1"/>
   <xsl:param name="arg2"/>
   <xsl:param name="arg3"/>
   <xsl:param name="arg4"/>
   <xsl:param name="arg5"/>
   
    <xsl:apply-templates select="$pFunc" mode="f:FXSL">
      <xsl:with-param name="arg1" select="$arg1"/>
      <xsl:with-param name="arg2" select="$arg2"/>
      <xsl:with-param name="arg3" select="$arg3"/>
      <xsl:with-param name="arg4" select="$arg4"/>
      <xsl:with-param name="arg5" select="$arg5"/>
    </xsl:apply-templates>
  </xsl:function>

  <xsl:function name="f:apply">
   <xsl:param name="pFunc" as="element()"/>
   <xsl:param name="arg1"/>
   <xsl:param name="arg2"/>
   <xsl:param name="arg3"/>
   <xsl:param name="arg4"/>
   <xsl:param name="arg5"/>
   <xsl:param name="arg6"/>
   
    <xsl:apply-templates select="$pFunc" mode="f:FXSL">
      <xsl:with-param name="arg1" select="$arg1"/>
      <xsl:with-param name="arg2" select="$arg2"/>
      <xsl:with-param name="arg3" select="$arg3"/>
      <xsl:with-param name="arg4" select="$arg4"/>
      <xsl:with-param name="arg5" select="$arg5"/>
      <xsl:with-param name="arg6" select="$arg6"/>
    </xsl:apply-templates>
  </xsl:function>

  <xsl:function name="f:apply">
   <xsl:param name="pFunc" as="element()"/>
   <xsl:param name="arg1"/>
   <xsl:param name="arg2"/>
   <xsl:param name="arg3"/>
   <xsl:param name="arg4"/>
   <xsl:param name="arg5"/>
   <xsl:param name="arg6"/>
   <xsl:param name="arg7"/>
   
    <xsl:apply-templates select="$pFunc" mode="f:FXSL">
      <xsl:with-param name="arg1" select="$arg1"/>
      <xsl:with-param name="arg2" select="$arg2"/>
      <xsl:with-param name="arg3" select="$arg3"/>
      <xsl:with-param name="arg4" select="$arg4"/>
      <xsl:with-param name="arg5" select="$arg5"/>
      <xsl:with-param name="arg6" select="$arg6"/>
      <xsl:with-param name="arg7" select="$arg7"/>
    </xsl:apply-templates>
  </xsl:function>

  <xsl:function name="f:apply">
   <xsl:param name="pFunc" as="element()"/>
   <xsl:param name="arg1"/>
   <xsl:param name="arg2"/>
   <xsl:param name="arg3"/>
   <xsl:param name="arg4"/>
   <xsl:param name="arg5"/>
   <xsl:param name="arg6"/>
   <xsl:param name="arg7"/>
   <xsl:param name="arg8"/>
   
    <xsl:apply-templates select="$pFunc" mode="f:FXSL">
      <xsl:with-param name="arg1" select="$arg1"/>
      <xsl:with-param name="arg2" select="$arg2"/>
      <xsl:with-param name="arg3" select="$arg3"/>
      <xsl:with-param name="arg4" select="$arg4"/>
      <xsl:with-param name="arg5" select="$arg5"/>
      <xsl:with-param name="arg6" select="$arg6"/>
      <xsl:with-param name="arg7" select="$arg7"/>
      <xsl:with-param name="arg8" select="$arg8"/>
    </xsl:apply-templates>
  </xsl:function>

  <xsl:function name="f:apply">
   <xsl:param name="pFunc" as="element()"/>
   <xsl:param name="arg1"/>
   <xsl:param name="arg2"/>
   <xsl:param name="arg3"/>
   <xsl:param name="arg4"/>
   <xsl:param name="arg5"/>
   <xsl:param name="arg6"/>
   <xsl:param name="arg7"/>
   <xsl:param name="arg8"/>
   <xsl:param name="arg9"/>
   
    <xsl:apply-templates select="$pFunc" mode="f:FXSL">
      <xsl:with-param name="arg1" select="$arg1"/>
      <xsl:with-param name="arg2" select="$arg2"/>
      <xsl:with-param name="arg3" select="$arg3"/>
      <xsl:with-param name="arg4" select="$arg4"/>
      <xsl:with-param name="arg5" select="$arg5"/>
      <xsl:with-param name="arg6" select="$arg6"/>
      <xsl:with-param name="arg7" select="$arg7"/>
      <xsl:with-param name="arg8" select="$arg8"/>
      <xsl:with-param name="arg9" select="$arg9"/>
    </xsl:apply-templates>
  </xsl:function>

  <xsl:function name="f:apply">
   <xsl:param name="pFunc" as="element()"/>
   <xsl:param name="arg1"/>
   <xsl:param name="arg2"/>
   <xsl:param name="arg3"/>
   <xsl:param name="arg4"/>
   <xsl:param name="arg5"/>
   <xsl:param name="arg6"/>
   <xsl:param name="arg7"/>
   <xsl:param name="arg8"/>
   <xsl:param name="arg9"/>
   <xsl:param name="arg10"/>
   
    <xsl:apply-templates select="$pFunc" mode="f:FXSL">
      <xsl:with-param name="arg1" select="$arg1"/>
      <xsl:with-param name="arg2" select="$arg2"/>
      <xsl:with-param name="arg3" select="$arg3"/>
      <xsl:with-param name="arg4" select="$arg4"/>
      <xsl:with-param name="arg5" select="$arg5"/>
      <xsl:with-param name="arg6" select="$arg6"/>
      <xsl:with-param name="arg7" select="$arg7"/>
      <xsl:with-param name="arg8" select="$arg8"/>
      <xsl:with-param name="arg9" select="$arg9"/>
      <xsl:with-param name="arg10" select="$arg10"/>
    </xsl:apply-templates>
  </xsl:function>
  
  <xsl:function name="f:drop" as="item()*">
    <xsl:param name="pN" as="xs:integer"/>
    <xsl:param name="pList" as="item()*"/>
    
    <xsl:sequence select="$pList[position() > $pN]"/>
  </xsl:function>  
  


    <xsl:function name="f:foldl">
      <xsl:param name="pFunc" as="element()"/>
      <xsl:param name="pA0"/>
      <xsl:param name="pList" as="item()*"/>

      <xsl:sequence select=
       "if(empty($pList))
          then 
             $pA0
          else
             for $vcntList in count($pList) return
               if($vcntList = 1)
                  then
                     f:apply($pFunc, $pA0, $pList[1])
                  else
                     for $vHalfLen in ($vcntList idiv 2) return
                       f:foldl($pFunc, 
                             f:foldl($pFunc, $pA0, $pList[position() le $vHalfLen]),
                             f:drop($vHalfLen, $pList)
                             )"/>

    </xsl:function>

    <xsl:function name="f:maximum" as="item()">
      <xsl:param name="pList" as="item()+"/>
      <xsl:param name="pCMPFun" as="element()"/>

      <xsl:sequence select="f:foldl($pCMPFun, $pList[1], $pList)"/>
    </xsl:function>

    <xsl:function name="f:maximum" as="item()">
      <xsl:param name="pList" as="item()+"/>
      <!--<xsl:message> 
        $pList: '<xsl:copy-of select="$pList"/>'
      </xsl:message>-->      
      <xsl:variable name="vCMPFun" as="element()">
        <maximum-pick-bigger:maximum-pick-bigger/>
      </xsl:variable> 

      <xsl:sequence select="f:maximum($pList, $vCMPFun)"/>
    </xsl:function>

    
    <xsl:template name="pickBigger" match="maximum-pick-bigger:*"
     mode="f:FXSL">
      <xsl:param name="arg1"/>
      <xsl:param name="arg2"/>

      <xsl:sequence select="if($arg1 > $arg2)
                              then $arg1
                              else $arg2"/>
    </xsl:template>
    
     <Mycompare:Mycompare/>

    <xsl:template match="/">

      <result><xsl:copy-of select="f:maximum(1 to 10)"/></result>

    </xsl:template>

    <xsl:template name="MySelectGreater" mode="f:FXSL"
     match="*[namespace-uri() = 'Mycompare']">
         <xsl:param name="arg1"/>
         <xsl:param name="arg2"/>

      <xsl:sequence select="if($arg1 &lt; $arg2)
                              then $arg1
                              else $arg2"/>
    </xsl:template>    
</xsl:stylesheet>