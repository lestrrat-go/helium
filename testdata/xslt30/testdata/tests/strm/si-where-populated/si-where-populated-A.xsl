<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:map="http://www.w3.org/2005/xpath-functions/map"
  xmlns:err="http://www.w3.org/2005/xqt-errors" xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="map xs err">

  <xsl:variable name="RUN" select="true()" static="yes"/>
  <xsl:strip-space elements="*"/>

 
  <!-- Test of xsl:where-populated with xsl:element (not empty) -->

  <xsl:template name="s-001" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:where-populated>
          <xsl:element name="a">
            <xsl:sequence select="./BOOKLIST/BOOKS/ITEM/PRICEDATA"/>
          </xsl:element>
        </xsl:where-populated>  
      </out>
    </xsl:source-document>
  </xsl:template>
  
  <!-- Test of xsl:where-populated with multiple elements, some empty -->
  
  <xsl:template name="s-002" use-when="$RUN">
       <xsl:source-document streamable="yes" href="../docs/books.xml">
           <Results>
               <xsl:where-populated>
                 <first>one</first>
                 <second/>
                 <xsl:copy-of select="BOOKLIST/CATEGORIES/*" />
                 <last>last</last>
               </xsl:where-populated>
           </Results>
       </xsl:source-document>
   </xsl:template> 
   
   <!-- Test of xsl:where-populated with atomic values, some empty -->
  
  <xsl:template name="s-003" use-when="$RUN">
       <xsl:source-document streamable="yes" href="../docs/books.xml">
           <out>
               <xsl:where-populated>
                 <xsl:sequence select="23, '', xs:date('2011-01-01'), xs:untypedAtomic(''), 0, /JUNK, xs:base64Binary('')"/>
               </xsl:where-populated>
           </out>
       </xsl:source-document>
   </xsl:template>
   
   <!-- Test of xsl:where-populated with xsl:if -->

  <xsl:template name="s-004" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:where-populated>
          <a>
            <xsl:if test="current-date() lt xs:date('1900-01-01')">
               <banana x="{count(//*)}"/>
            </xsl:if>
          </a>
        </xsl:where-populated>  
      </out>
    </xsl:source-document>
  </xsl:template> 
  
  <!-- Test of xsl:where-populated with xsl:if -->

  <xsl:template name="s-005" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:where-populated>
          <a>
            <xsl:if test="current-date() gt xs:date('1900-01-01')">
               <banana x="{count(//*)}"/>
            </xsl:if>
          </a>
        </xsl:where-populated>  
      </out>
    </xsl:source-document>
  </xsl:template> 
  
  <!-- Test of xsl:where-populated with xsl:for-each -->

  <xsl:template name="s-006" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:where-populated>
          <ul>
            <xsl:for-each select="outermost(//ITEM)">
              <li><xsl:value-of select="TITLE"/></li>
            </xsl:for-each>  
          </ul>
        </xsl:where-populated>  
      </out>
    </xsl:source-document>
  </xsl:template> 
  
  <!-- Test of xsl:where-populated with xsl:for-each -->

  <xsl:template name="s-007" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:where-populated>
          <ul>
            <xsl:for-each select="outermost(//MAGAZINE)">
              <li><xsl:value-of select="TITLE"/></li>
            </xsl:for-each>  
          </ul>
        </xsl:where-populated>  
      </out>
    </xsl:source-document>
  </xsl:template> 
  
  <!-- Test of xsl:where-populated with comment nodes -->

  <xsl:template name="s-008" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:for-each select="outermost(//ITEM)">
          <in>
            <xsl:where-populated>
              <xsl:comment select="TITLE[parent::BOOK]"/>
            </xsl:where-populated>
            <xsl:where-populated>
              <xsl:comment select="count(ancestor::node())"/>
            </xsl:where-populated> 
          </in>  
         </xsl:for-each>  
      </out>
    </xsl:source-document>
  </xsl:template> 
  
  <!-- Test of xsl:where-populated with text nodes -->

  <xsl:template name="s-009" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:for-each select="outermost(//ITEM)">
          <in>
            <xsl:where-populated>
              <xsl:value-of select="TITLE[parent::BOOK]"/>
            </xsl:where-populated>
            <xsl:where-populated>
              <xsl:value-of select="count(ancestor::node())"/>
            </xsl:where-populated> 
          </in>  
         </xsl:for-each>  
      </out>
    </xsl:source-document>
  </xsl:template> 
  
  <!-- Test of xsl:where-populated with xsl:fork -->

  <xsl:template name="s-010" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:where-populated>
          <a>
            <xsl:fork>
              <xsl:sequence select="copy-of(//TITLE)"/>
              <xsl:sequence select="copy-of(//PRICE)"/>  
            </xsl:fork>
          </a>
        </xsl:where-populated>  
      </out>
    </xsl:source-document>
  </xsl:template> 
  
  <!-- Test of xsl:where-populated with xsl:fork -->

  <xsl:template name="s-011" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:where-populated>
          <a>
            <xsl:fork>
              <xsl:sequence select="copy-of(//TITTLE)"/>
              <xsl:sequence select="copy-of(//PRICLE)"/>  
            </xsl:fork>
          </a>
        </xsl:where-populated>  
      </out>
    </xsl:source-document>
  </xsl:template>
  
  <!-- Test of xsl:where-populated with document nodes -->
  
  <xsl:template name="s-012" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:variable name="t" as="document-node()?">          
          <xsl:where-populated>
            <xsl:document>
              <xsl:copy-of select="//TITLE[@flamingo]"/>
            </xsl:document>
          </xsl:where-populated>
        </xsl:variable>
        <in value="{empty($t)}"/>     
      </out>
    </xsl:source-document>
  </xsl:template> 
  
  <!-- Test of xsl:where-populated with maps (bug 28989) -->
  
  <xsl:template name="s-013" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:variable name="m" as="map(*)?">          
          <xsl:where-populated>
            <!-- empty map -->
            <xsl:map>
              <xsl:for-each select="/*/*/QUARGLE">
                <xsl:map-entry key="@id" select="data(@pongle)"/>
              </xsl:for-each>
            </xsl:map>
          </xsl:where-populated>
        </xsl:variable>
        <in count="{count($m)}"/>     
      </out>
    </xsl:source-document>
  </xsl:template> 
  
  <!-- Test of xsl:where-populated with maps (bug 28989) -->
  
  <xsl:template name="s-014" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:variable name="m" as="map(*)?">          
          <xsl:where-populated>
            <!-- non-empty map -->
            <xsl:map>
              <xsl:for-each select="/*/*/ITEM">
                <xsl:map-entry key="generate-id()" select="copy-of(TITLE)"/>
              </xsl:for-each>
            </xsl:map>
          </xsl:where-populated>
        </xsl:variable>
        <in count="{count($m)}"/>     
      </out>
    </xsl:source-document>
  </xsl:template> 
 
  <!-- Test of xsl:where-populated with elements having attributes and namespaces -->
  
  <xsl:template name="s-015" use-when="$RUN">
       <xsl:source-document streamable="yes" href="../docs/books.xml">
           <Results>
               <xsl:where-populated>
                 <first one='non-empty'>one</first>
                 <second one='empty'/>
                 <third xmlns:x='http://abc.com/'>three</third>
                 <fourth xmlns:x='http://abc.com/'/>
               </xsl:where-populated>
           </Results>
       </xsl:source-document>
   </xsl:template>  
   
   <!-- Test of xsl:where-populated with free-standing attributes -->
  
  <xsl:template name="s-016" use-when="$RUN">
       <xsl:source-document streamable="yes" href="../docs/books.xml">
           <Results>
               <xsl:where-populated>
                 <xsl:attribute name="x" select="17"/>
                 <xsl:attribute name="x" select="''"/>
                 <inner/>
                 <xsl:attribute name="y" select="substring(string(current-date()), 842)"/>
               </xsl:where-populated>
           </Results>
       </xsl:source-document>
   </xsl:template>  
   
   <!-- A namespace node with zero-length string value is an error even within where-populated -->
  
  <xsl:template name="s-017" use-when="$RUN">
       <xsl:source-document streamable="yes" href="../docs/books.xml">
           <Results>
               <xsl:where-populated>
                 <xsl:namespace name="x" select="substring(string(current-date()), 842)"/>
               </xsl:where-populated>
           </Results>
       </xsl:source-document>
   </xsl:template>
   
   <!-- Test of xsl:where-populated with zero-length text nodes. Note that these are
   eliminated before the "constructing complex content" rules come into play, so atomic values
   either side become adjacent. -->
  
  <xsl:template name="s-018" use-when="$RUN">
       <xsl:source-document streamable="yes" href="../docs/books.xml">
           <Results>
               <xsl:where-populated>
                 <xsl:sequence select="17"/>
                 <xsl:value-of select="//yankee-doodle"/>
                 <xsl:sequence select="92"/>
                 <xsl:text/>
                 <xsl:sequence select="55"/>
                 <xsl:text>=</xsl:text>
                 <xsl:value-of>17 92 55</xsl:value-of>
               </xsl:where-populated>
           </Results>
       </xsl:source-document>
   </xsl:template>  
   
   <!-- Test of xsl:where-populated with zero-length comments. -->
  
  <xsl:template name="s-019" use-when="$RUN">
       <xsl:source-document streamable="yes" href="../docs/books.xml">
           <Results>
               <xsl:where-populated>
                 <xsl:text>1:</xsl:text>
                 <xsl:comment select="head(//TITLE)"/>
                 <xsl:text>2:</xsl:text>
                 <xsl:comment select="unparsed-entity-uri('fandango')"/>
               </xsl:where-populated>
           </Results>
       </xsl:source-document>
   </xsl:template>     
   
   <!-- Test of xsl:where-populated with zero-length processing-instructions. -->
  
  <xsl:template name="s-020" use-when="$RUN">
       <xsl:source-document streamable="yes" href="../docs/books.xml">
           <Results>
               <xsl:where-populated>
                 <xsl:text>1:</xsl:text>
                 <xsl:processing-instruction name="mu" select="head(//TITLE)"/>
                 <xsl:text>2:</xsl:text>
                 <xsl:processing-instruction name="pi" select="unparsed-entity-uri('fandango')"/>
               </xsl:where-populated>
           </Results>
       </xsl:source-document>
   </xsl:template>
   
   <!-- Test of xsl:where-populated including local variables. -->
  
  <xsl:template name="s-021" use-when="$RUN">
       <xsl:source-document streamable="yes" href="../docs/books.xml">
           <Results>
               <xsl:where-populated>
                 <xsl:variable name="x"/>
                 <xsl:variable name="y" select="22"/>
                 <a><xsl:value-of select="$x||$x"/></a>
                 <b><xsl:value-of select="$y||$y"/></b>
                 <xsl:variable name="xx"/>
                 <xsl:variable name="yy" select="22"/>
                 <a><xsl:value-of select="$xx||$xx"/></a>
                 <b><xsl:value-of select="$yy||$yy"/></b>
               </xsl:where-populated>
           </Results>
       </xsl:source-document>
   </xsl:template>         
 




</xsl:stylesheet>
