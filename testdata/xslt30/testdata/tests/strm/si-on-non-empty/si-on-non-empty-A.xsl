<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:map="http://www.w3.org/2005/xpath-functions/map"
  xmlns:err="http://www.w3.org/2005/xqt-errors" xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="map xs err">

  <xsl:variable name="RUN" select="true()" static="yes"/>
  <!-- Parameters used to inhibit static optimizations -->
  <xsl:param name="empty" select="()"/>
  <xsl:param name="param-string" select="''"/>
  <xsl:param name="underscore" select="'_'"/>
  <xsl:strip-space elements="*"/>

 
  
  
  <!-- Test of xsl:on-non-empty with xsl:element (not empty) -->

  <xsl:template name="s-001" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:sequence select="$param-string"/>
        <xsl:on-non-empty>
          <xsl:element name="a">
            <xsl:sequence select="./BOOKLIST/BOOKS/ITEM/PRICEDATA"/>
          </xsl:element>
        </xsl:on-non-empty>  
      </out>
    </xsl:source-document>
  </xsl:template>
  
  <!-- Test of xsl:on-non-empty with multiple elements, some empty -->
  
  <xsl:template name="s-002" use-when="$RUN">
       <xsl:source-document streamable="yes" href="../docs/books.xml">
           <Results>
               <xsl:sequence select="$param-string"/>
               <xsl:on-non-empty>
                 <first>one</first>
                 <second/>
                 <xsl:copy-of select="BOOKLIST/CATEGORIES/*" />
                 <last>last</last>
               </xsl:on-non-empty>
           </Results>
       </xsl:source-document>
   </xsl:template> 
   
   <!-- Test of xsl:on-non-empty with atomic values, some empty -->
  
  <xsl:template name="s-003" use-when="$RUN">
       <xsl:source-document streamable="yes" href="../docs/books.xml">
           <out>
               <xsl:sequence select="$param-string"/>
               <xsl:on-non-empty>
                 <xsl:sequence select="23, '', xs:date('2011-01-01'), xs:untypedAtomic(''), 0, /JUNK, xs:base64Binary('')"/>
               </xsl:on-non-empty>
           </out>
       </xsl:source-document>
   </xsl:template>
   
   <!-- Test of xsl:on-non-empty with xsl:if -->

  <xsl:template name="s-004" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:sequence select="$param-string"/>
        <xsl:on-non-empty>
          <a>
            <xsl:if test="current-date() lt xs:date('1900-01-01')">
               <banana x="{count(//*)}"/>
            </xsl:if>
          </a>
        </xsl:on-non-empty>  
      </out>
    </xsl:source-document>
  </xsl:template> 
  
  <!-- Test of xsl:on-non-empty with xsl:if -->

  <xsl:template name="s-005" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:sequence select="$param-string"/>
        <xsl:on-non-empty>
          <a>
            <xsl:if test="current-date() gt xs:date('1900-01-01')">
               <banana x="{count(//*)}"/>
            </xsl:if>
          </a>
        </xsl:on-non-empty>  
      </out>
    </xsl:source-document>
  </xsl:template> 
  
  <!-- Test of xsl:on-non-empty with xsl:for-each -->

  <xsl:template name="s-006" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:sequence select="$param-string"/>
        <xsl:on-non-empty>
          <ul>
            <xsl:for-each select="outermost(//ITEM)">
              <li><xsl:value-of select="TITLE"/></li>
            </xsl:for-each>  
          </ul>
        </xsl:on-non-empty>  
      </out>
    </xsl:source-document>
  </xsl:template> 
  
  <!-- Test of xsl:on-non-empty with xsl:for-each -->

  <xsl:template name="s-007" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:sequence select="$param-string"/>
        <xsl:on-non-empty>
          <ul>
            <xsl:for-each select="outermost(//MAGAZINE)">
              <li><xsl:value-of select="TITLE"/></li>
            </xsl:for-each>  
          </ul>
        </xsl:on-non-empty>  
      </out>
    </xsl:source-document>
  </xsl:template> 
  
  <!-- Test of xsl:on-non-empty with comment nodes -->

  <xsl:template name="s-008" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:for-each select="outermost(//ITEM)">
          <in>
            <xsl:sequence select="$param-string"/>
            <xsl:on-non-empty>
              <xsl:comment select="TITLE[parent::BOOK]"/>
            </xsl:on-non-empty>
            <xsl:on-non-empty>
              <xsl:comment select="count(ancestor::node())"/>
            </xsl:on-non-empty> 
          </in>  
         </xsl:for-each>  
      </out>
    </xsl:source-document>
  </xsl:template> 
  
  <!-- Test of xsl:on-non-empty with text nodes -->

  <xsl:template name="s-009" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:for-each select="outermost(//ITEM)">
          <in>
            <xsl:sequence select="$param-string"/>
            <xsl:on-non-empty>
              <xsl:value-of select="TITLE[parent::BOOK]"/>
            </xsl:on-non-empty>
            <xsl:on-non-empty>
              <xsl:value-of select="count(ancestor::node())"/>
            </xsl:on-non-empty> 
          </in>  
         </xsl:for-each>  
      </out>
    </xsl:source-document>
  </xsl:template> 
  
  <!-- Test of xsl:on-non-empty with xsl:fork -->

  <xsl:template name="s-010" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:sequence select="$underscore"/>
        <xsl:on-non-empty>
          <a>
            <xsl:fork>
              <xsl:sequence select="copy-of(//TITLE)"/>
              <xsl:sequence select="copy-of(//PRICE)"/>  
            </xsl:fork>
          </a>
        </xsl:on-non-empty>  
      </out>
    </xsl:source-document>
  </xsl:template> 
  
  <!-- Test of xsl:on-non-empty with xsl:fork -->

  <xsl:template name="s-011" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:sequence select="$param-string"/>
        <xsl:on-non-empty>
          <a>
            <xsl:fork>
              <xsl:sequence select="copy-of(//TITTLE)"/>
              <xsl:sequence select="copy-of(//PRICLE)"/>  
            </xsl:fork>
          </a>
        </xsl:on-non-empty>  
      </out>
    </xsl:source-document>
  </xsl:template>
  
  <!-- Test of xsl:on-non-empty with document nodes -->
  
  <xsl:template name="s-012" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:variable name="t" as="document-node()?">          
          <xsl:on-non-empty>
            <xsl:document>
              <xsl:copy-of select="//TITLE[@flamingo]"/>
            </xsl:document>
          </xsl:on-non-empty>
        </xsl:variable>
        <in value="{empty($t)}"/>     
      </out>
    </xsl:source-document>
  </xsl:template> 
 
  <!-- Test of xsl:on-non-empty with xsl:element (empty) -->
  
  <xsl:template name="s-021" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:element name="a">
          <xsl:sequence select="./BOOKLIST/BOOKS/ITEM/PRICEDATA"/>
          <xsl:on-non-empty>There is no price data</xsl:on-non-empty>
        </xsl:element>         
      </out>
    </xsl:source-document>
  </xsl:template>
  
  <!-- Test of xsl:on-non-empty with xsl:element (not empty) -->
  
  <xsl:template name="s-022" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:element name="a">
          <xsl:sequence select="./BOOKLIST/BOOKS/ITEM/PRICE"/>
          <xsl:on-non-empty>There is some price data</xsl:on-non-empty>
        </xsl:element>         
      </out>
    </xsl:source-document>
  </xsl:template>
  
  <!-- Test of xsl:on-non-empty with xsl:element (empty) -->
  
  <xsl:template name="s-023" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:element name="a">
          <xsl:on-non-empty>There is no price data</xsl:on-non-empty>
          <xsl:sequence select="./BOOKLIST/BOOKS/ITEM/PRICEDATA"/>         
        </xsl:element>         
      </out>
    </xsl:source-document>
  </xsl:template>
  
  <!-- Test of xsl:on-non-empty with xsl:element (not empty) -->
  
  <xsl:template name="s-024" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:element name="a">
          <xsl:on-non-empty>There is some price data</xsl:on-non-empty>
          <xsl:sequence select="./BOOKLIST/BOOKS/ITEM/PRICE"/>         
        </xsl:element>         
      </out>
    </xsl:source-document>
  </xsl:template>
  
  <!-- Test of xsl:on-non-empty with multiple elements, some empty -->
  
  <xsl:template name="s-025" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:sequence select="./BOOKLIST/BOOKS/ITEM/PRICEDATA"/>  
        <xsl:on-non-empty>There is some price data</xsl:on-non-empty>
        <xsl:sequence select="ends-with(base-uri(), 'books.xml')[exists($param-string)]"/>    
      </out>
    </xsl:source-document>
  </xsl:template> 
  
  <!-- Test of xsl:on-non-empty with multiple elements, some empty -->
  
  <xsl:template name="s-026" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:sequence select="ends-with(base-uri(), 'books.xml')[exists($param-string)]"/>  
        <xsl:on-non-empty>There is some price data</xsl:on-non-empty>
        <xsl:sequence select="./BOOKLIST/BOOKS/ITEM/PRICEDATA"/>    
      </out>
    </xsl:source-document>
  </xsl:template> 
  
  <!-- Test of xsl:on-non-empty with multiple elements, all empty -->
  
  <xsl:template name="s-027" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:for-each select="BOOKLIST">
          <xsl:sequence select="data(@dummy)"/>  
          <xsl:on-non-empty>There is some price data</xsl:on-non-empty>
          <xsl:sequence select="BOOKS/ITEM/PRICEDATA"/> 
        </xsl:for-each>
      </out>
    </xsl:source-document>
  </xsl:template> 
  
  <!-- Test of xsl:on-non-empty with atomic values, some empty -->
  
  <xsl:template name="s-028" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:sequence select="(1 to 20)[. ge 20][exists($param-string)]"/>  
        <xsl:on-non-empty>There is some price data</xsl:on-non-empty>
        <xsl:sequence select="./BOOKLIST/BOOKS/ITEM/PRICEDATA/data()"/> 
      </out>
    </xsl:source-document>
  </xsl:template>
  
  <!-- Test of xsl:on-non-empty with atomic values, some empty -->
  
  <xsl:template name="s-029" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:sequence select="(1 to 20)[. ge 40]"/>  
        <xsl:on-non-empty>There is some price data</xsl:on-non-empty>
        <xsl:sequence select="./BOOKLIST/BOOKS/ITEM/PRICE/data()"/> 
      </out>
    </xsl:source-document>
  </xsl:template>
  
  <!-- Test of xsl:on-non-empty with xsl:variable -->
  
  <xsl:template name="s-030" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <xsl:for-each select="BOOKLIST">
        <out>
          <xsl:variable name="x" select="name()"/>
          <xsl:sequence select="$param-string"/>
          <xsl:on-non-empty>There is no price data within <xsl:value-of select="$x"/></xsl:on-non-empty> 
        </out>
      </xsl:for-each>
    </xsl:source-document>
  </xsl:template>
  
  <!-- Test of xsl:on-non-empty with xsl:variable -->
  
  <xsl:template name="s-031" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <xsl:for-each select="BOOKLIST">
        <out>
          <xsl:sequence select="BOOKS/ITEM/PRICEDATA"/>
          <xsl:variable name="x" select="name()"/>
          <xsl:on-non-empty>There is no price data within <xsl:value-of select="$x"/></xsl:on-non-empty>
        </out>
      </xsl:for-each>
    </xsl:source-document>
  </xsl:template>
  
  <!-- Test of xsl:on-non-empty with xsl:if -->
  
  <xsl:template name="s-032" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
          <xsl:if test="current-date() lt xs:date('1900-01-01')">
            <banana x="{count(//*)}"/>
          </xsl:if>
          <xsl:on-non-empty>Howdy!</xsl:on-non-empty>  
      </out>
    </xsl:source-document>
  </xsl:template> 
  
  <!-- Test of xsl:on-non-empty with xsl:if -->
  
  <xsl:template name="s-033" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
          <xsl:if test="current-date() gt xs:date('1900-01-01')">
            <banana x="{count(//*)}"/>
          </xsl:if>
          <xsl:on-non-empty>Howdy!</xsl:on-non-empty>       
      </out>
    </xsl:source-document>
  </xsl:template> 
  
  <!-- Test of xsl:on-non-empty with xsl:for-each and xsl:where-populated -->
  
  <xsl:template name="s-034" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:where-populated>
          <ul>
            <xsl:for-each select="outermost(//ITEM)">
              <xsl:where-populated>
                <li><xsl:value-of select="TITTLE-TATTLE"/></li>
              </xsl:where-populated>
            </xsl:for-each>  
          </ul>
        </xsl:where-populated>
        <xsl:on-non-empty>There has been no gossip.</xsl:on-non-empty>
      </out>
    </xsl:source-document>
  </xsl:template> 
  
  <!-- Test of xsl:on-non-empty with xsl:for-each and xsl:where-populated -->
  
  <xsl:template name="s-035" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:on-non-empty>There will be no gossip.</xsl:on-non-empty>
        <xsl:where-populated>
          <ul>
            <xsl:for-each select="outermost(//ITEM)">
              <xsl:where-populated>
                <li><xsl:value-of select="TITTLE-TATTLE"/></li>
              </xsl:where-populated>
            </xsl:for-each>  
          </ul>
        </xsl:where-populated>
      </out>
    </xsl:source-document>
  </xsl:template> 
  
  <!-- Test of xsl:on-non-empty (appearing twice) with xsl:for-each and xsl:where-populated -->
  
  <xsl:template name="s-036" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:on-non-empty>There will be no gossip.</xsl:on-non-empty>
        <xsl:where-populated>
          <ul>
            <xsl:for-each select="outermost(//ITEM)">
              <xsl:where-populated>
                <li><xsl:value-of select="TITTLE-TATTLE"/></li>
              </xsl:where-populated>
            </xsl:for-each>  
          </ul>
        </xsl:where-populated>
        <xsl:on-non-empty>There has been no gossip.</xsl:on-non-empty>
      </out>
    </xsl:source-document>
  </xsl:template> 
  
  
  <!-- Test of xsl:on-non-empty with xsl:fork -->
  
  <xsl:template name="s-037" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <a>
          <xsl:fork>
            <xsl:sequence select="copy-of(//TITLE)"/>
            <xsl:sequence select="copy-of(//PRICE)"/>  
          </xsl:fork>
          <xsl:on-non-empty>The forks were not empty.</xsl:on-non-empty>
        </a>        
      </out>
    </xsl:source-document>
  </xsl:template> 
  
  <!-- Test of xsl:on-non-empty with xsl:fork -->
  
  <xsl:template name="s-038" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <a>
          <xsl:fork>
            <xsl:sequence select="copy-of(//TITTLE)"/>
            <xsl:sequence select="copy-of(//PRICLE)"/>  
          </xsl:fork>
          <xsl:on-non-empty>The forks were not empty.</xsl:on-non-empty>
        </a>        
      </out>
    </xsl:source-document>
  </xsl:template> 
  
  <!-- Test of consuming xsl:on-non-empty (executed) -->
  
  <xsl:template name="s-039" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:if test="current-date() lt xs:date('1900-01-01')">
          <content/>
        </xsl:if>
        <xsl:on-non-empty>
          <xsl:for-each select="/BOOKLIST/BOOKS/ITEM">
            <xsl:copy-of select="TITLE"/>
          </xsl:for-each>
        </xsl:on-non-empty>
      </out>
    </xsl:source-document>
  </xsl:template> 
  
  <!-- Test of consuming xsl:on-non-empty (executed) -->
  
  <xsl:template name="s-040" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:if test="current-date() gt xs:date('1900-01-01')">
          <content/>
        </xsl:if>
        <xsl:on-non-empty>
          <xsl:for-each select="/BOOKLIST/BOOKS/ITEM">
            <xsl:copy-of select="TITLE"/>
          </xsl:for-each>
        </xsl:on-non-empty>
      </out>
    </xsl:source-document>
  </xsl:template> 
  
  <!-- Zero length text nodes are considered empty -->
  
  <xsl:template name="s-041" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:value-of select="string(/BOOKLIST/BOOKS/ITEM/EXPORT-QUOTA)"/>
        <xsl:on-non-empty>WRONG</xsl:on-non-empty>
     </out>
    </xsl:source-document>
  </xsl:template> 
  
  <!-- Childless document nodes are considered empty -->
  
  <xsl:template name="s-042" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:document>
          <xsl:value-of select="string(/BOOKLIST/BOOKS/ITEM/EXPORT-QUOTA)"/>
        </xsl:document>  
        <xsl:on-non-empty>WRONG</xsl:on-non-empty>
     </out>
    </xsl:source-document>
  </xsl:template> 
  
  <!-- Document nodes containing whitespace text are not considered empty -->
  
  <xsl:template name="s-043" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:on-non-empty>*</xsl:on-non-empty>
        <xsl:document>
          <xsl:value-of select="concat(' ', string(/BOOKLIST/BOOKS/ITEM/EXPORT-QUOTA), ' ')"/>
        </xsl:document>  
        <xsl:on-non-empty>*</xsl:on-non-empty>
     </out>
    </xsl:source-document>
  </xsl:template> 
  
  <!-- Separators between zero-length strings do not make the content non-empty -->
  
  <xsl:template name="s-044" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:on-non-empty>*</xsl:on-non-empty>
        <xsl:for-each select="outermost(//PRICE)">
          <xsl:sequence select="substring(., 1000)"/>
        </xsl:for-each>
        <xsl:on-non-empty>*</xsl:on-non-empty>
        <xsl:on-empty>+++</xsl:on-empty>
      </out>
    </xsl:source-document>
  </xsl:template> 
  
  




</xsl:stylesheet>
