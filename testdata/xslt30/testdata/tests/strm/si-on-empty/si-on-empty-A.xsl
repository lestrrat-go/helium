<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:map="http://www.w3.org/2005/xpath-functions/map"
  xmlns:err="http://www.w3.org/2005/xqt-errors" xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="map xs err">

  <xsl:variable name="RUN" select="true()" static="yes"/>
  <xsl:param name="empty" select="()"/> <!-- to prevent static optimization -->
  <xsl:strip-space elements="*"/>

 
  <!-- Note, the first few tests are mechanically copied from xsl:where-populated
       tests. They don't do anything meaningful, but are retained to ensure the code
       paths work. -->
        
 
  
  <!-- Test of xsl:on-empty with xsl:element (not empty) -->

  <xsl:template name="s-001" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:sequence select="$empty"/>
        <xsl:on-empty>
          <xsl:element name="a">
            <xsl:sequence select="./BOOKLIST/BOOKS/ITEM/PRICEDATA"/>
          </xsl:element>
        </xsl:on-empty>  
      </out>
    </xsl:source-document>
  </xsl:template>
  
  <!-- Test of xsl:on-empty with multiple elements, some empty -->
  
  <xsl:template name="s-002" use-when="$RUN">
       <xsl:source-document streamable="yes" href="../docs/books.xml">
           <Results>
               <xsl:sequence select="$empty"/>
               <xsl:on-empty>
                 <first>one</first>
                 <second/>
                 <xsl:copy-of select="BOOKLIST/CATEGORIES/*" />
                 <last>last</last>
               </xsl:on-empty>
           </Results>
       </xsl:source-document>
   </xsl:template> 
   
   <!-- Test of xsl:on-empty with atomic values, some empty -->
  
  <xsl:template name="s-003" use-when="$RUN">
       <xsl:source-document streamable="yes" href="../docs/books.xml">
           <out>
               <xsl:sequence select="$empty"/>
               <xsl:on-empty>
                 <xsl:sequence select="23, '', xs:date('2011-01-01'), xs:untypedAtomic(''), 0, /JUNK, xs:base64Binary('')"/>
               </xsl:on-empty>
           </out>
       </xsl:source-document>
   </xsl:template>
   
   <!-- Test of xsl:on-empty with xsl:if -->

  <xsl:template name="s-004" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:sequence select="$empty"/>
        <xsl:on-empty>
          <a>
            <xsl:if test="current-date() lt xs:date('1900-01-01')">
               <banana x="{count(//*)}"/>
            </xsl:if>
          </a>
        </xsl:on-empty>  
      </out>
    </xsl:source-document>
  </xsl:template> 
  
  <!-- Test of xsl:on-empty with xsl:if -->

  <xsl:template name="s-005" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:sequence select="$empty"/>
        <xsl:on-empty>
          <a>
            <xsl:if test="current-date() gt xs:date('1900-01-01')">
               <banana x="{count(//*)}"/>
            </xsl:if>
          </a>
        </xsl:on-empty>  
      </out>
    </xsl:source-document>
  </xsl:template> 
  
  <!-- Test of xsl:on-empty with xsl:for-each -->

  <xsl:template name="s-006" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:sequence select="$empty"/>
        <xsl:on-empty>
          <ul>
            <xsl:for-each select="outermost(//ITEM)">
              <li><xsl:value-of select="TITLE"/></li>
            </xsl:for-each>  
          </ul>
        </xsl:on-empty>  
      </out>
    </xsl:source-document>
  </xsl:template> 
  
  <!-- Test of xsl:on-empty with xsl:for-each -->

  <xsl:template name="s-007" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:sequence select="$empty"/>
        <xsl:on-empty>
          <ul>
            <xsl:for-each select="outermost(//MAGAZINE)">
              <li><xsl:value-of select="TITLE"/></li>
            </xsl:for-each>  
          </ul>
        </xsl:on-empty>  
      </out>
    </xsl:source-document>
  </xsl:template> 
  
  <!-- Test of xsl:on-empty with comment nodes -->

  <xsl:template name="s-008" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:for-each select="outermost(//ITEM)">
          <in>
            <xsl:sequence select="$empty"/>
            <xsl:on-empty>
              <xsl:comment select="TITLE[parent::BOOK]"/>
              <xsl:comment select="count(ancestor::node())"/>
            </xsl:on-empty> 
          </in>  
         </xsl:for-each>  
      </out>
    </xsl:source-document>
  </xsl:template> 
  
  <!-- Test of xsl:on-empty with text nodes -->

  <xsl:template name="s-009" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:for-each select="outermost(//ITEM)">
          <in>
            <xsl:sequence select="$empty"/>
            <xsl:on-empty>
              <xsl:value-of select="TITLE[parent::BOOK]"/>
              <xsl:value-of select="count(ancestor::node())"/>
            </xsl:on-empty> 
          </in>  
         </xsl:for-each>  
      </out>
    </xsl:source-document>
  </xsl:template> 
  
  <!-- Test of xsl:on-empty with xsl:fork -->

  <xsl:template name="s-010" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:sequence select="$empty"/>
        <xsl:on-empty>
          <a>
            <xsl:fork>
              <xsl:sequence select="copy-of(//TITLE)"/>
              <xsl:sequence select="copy-of(//PRICE)"/>  
            </xsl:fork>
          </a>
        </xsl:on-empty>  
      </out>
    </xsl:source-document>
  </xsl:template> 
  
  <!-- Test of xsl:on-empty with xsl:fork -->

  <xsl:template name="s-011" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:sequence select="$empty"/>
        <xsl:on-empty>
          <a>
            <xsl:fork>
              <xsl:sequence select="copy-of(//TITTLE)"/>
              <xsl:sequence select="copy-of(//PRICLE)"/>  
            </xsl:fork>
          </a>
        </xsl:on-empty>  
      </out>
    </xsl:source-document>
  </xsl:template>
  
  <!-- Test of xsl:on-empty with document nodes -->
  
  <xsl:template name="s-012" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:variable name="t" as="document-node()?"> 
          <xsl:sequence select="$empty"/>         
          <xsl:on-empty>
            <xsl:where-populated>
              <xsl:document>
                <xsl:copy-of select="//TITLE[@flamingo]"/>
              </xsl:document>
            </xsl:where-populated>  
          </xsl:on-empty>
        </xsl:variable>
        <in value="{empty($t)}"/>     
      </out>
    </xsl:source-document>
  </xsl:template> 
 
  <!-- Test of xsl:on-empty with xsl:element (empty) -->
  
  <xsl:template name="s-021" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:element name="a">
          <xsl:sequence select="./BOOKLIST/BOOKS/ITEM/PRICEDATA"/>
          <xsl:on-empty>There is no price data</xsl:on-empty>
        </xsl:element>         
      </out>
    </xsl:source-document>
  </xsl:template>
  
  <!-- Test of xsl:on-empty with xsl:element (not empty) -->
  
  <xsl:template name="s-022" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:element name="a">
          <xsl:sequence select="./BOOKLIST/BOOKS/ITEM/PRICE"/>
          <xsl:on-empty>There is no price data</xsl:on-empty>
        </xsl:element>         
      </out>
    </xsl:source-document>
  </xsl:template>
  
  <!-- Test of xsl:on-empty with xsl:element (empty) -->
  
  <xsl:template name="s-023" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:element name="a">
          <xsl:sequence select="./BOOKLIST/BOOKS/ITEM/PRICEDATA"/> 
          <xsl:on-empty>There is no price data</xsl:on-empty>                  
        </xsl:element>         
      </out>
    </xsl:source-document>
  </xsl:template>
  
  <!-- Test of xsl:on-empty with xsl:element (not empty) -->
  
  <xsl:template name="s-024" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:element name="a">
          <xsl:sequence select="./BOOKLIST/BOOKS/ITEM/PRICE"/>
          <xsl:on-empty>There is no price data</xsl:on-empty>         
        </xsl:element>         
      </out>
    </xsl:source-document>
  </xsl:template>
  
  <!-- Test of xsl:on-empty with multiple elements, some empty -->
  
  <xsl:template name="s-025" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:sequence select="./BOOKLIST/BOOKS/ITEM/PRICEDATA"/>  
        <xsl:sequence select="ends-with(base-uri(), 'books.xml')[empty($empty)]"/>
        <xsl:on-empty>There is no price data</xsl:on-empty>    
      </out>
    </xsl:source-document>
  </xsl:template> 
  
  <!-- Test of xsl:on-empty with multiple elements, some empty -->
  
  <xsl:template name="s-026" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:sequence select="ends-with(base-uri(), 'books.xml')[empty($empty)]"/>  
        <xsl:sequence select="./BOOKLIST/BOOKS/ITEM/PRICEDATA"/>
        <xsl:on-empty>There is no price data</xsl:on-empty>    
      </out>
    </xsl:source-document>
  </xsl:template> 
  
  <!-- Test of xsl:on-empty with multiple elements, all empty -->
  
  <xsl:template name="s-027" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:for-each select="BOOKLIST">
          <xsl:sequence select="data(@dummy)"/>  
          <xsl:sequence select="BOOKS/ITEM/PRICEDATA"/>
          <xsl:on-empty>There is no price data</xsl:on-empty> 
        </xsl:for-each>
      </out>
    </xsl:source-document>
  </xsl:template> 
  
  <!-- Test of xsl:on-empty with atomic values, some empty -->
  
  <xsl:template name="s-028" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:sequence select="(1 to 20)[. ge 20][empty($empty)]"/>  
        <xsl:sequence select="./BOOKLIST/BOOKS/ITEM/PRICEDATA/data()"/>
        <xsl:on-empty>There is no price data</xsl:on-empty> 
      </out>
    </xsl:source-document>
  </xsl:template>
  
  <!-- Test of xsl:on-empty with atomic values, some empty -->
  
  <xsl:template name="s-029" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:sequence select="(1 to 20)[. ge 40]"/>  
        <xsl:sequence select="./BOOKLIST/BOOKS/ITEM/PRICE/data()"/>
        <xsl:on-empty>There is no price data</xsl:on-empty> 
      </out>
    </xsl:source-document>
  </xsl:template>
  
  <!-- Test of xsl:on-empty with xsl:variable -->
  
  <xsl:template name="s-030" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <xsl:for-each select="BOOKLIST">
        <out>
          <xsl:variable name="x" select="name()"/>
          <xsl:sequence select="$empty"/>
          <xsl:on-empty>There is no price data within <xsl:value-of select="$x"/></xsl:on-empty> 
        </out>
      </xsl:for-each>
    </xsl:source-document>
  </xsl:template>
  
  <!-- Test of xsl:on-empty with xsl:variable -->
  
  <xsl:template name="s-031" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <xsl:for-each select="BOOKLIST">
        <out>
          <xsl:sequence select="BOOKS/ITEM/PRICEDATA"/>
          <xsl:variable name="x" select="name()"/>
          <xsl:on-empty>There is no price data within <xsl:value-of select="$x"/></xsl:on-empty>
        </out>
      </xsl:for-each>
    </xsl:source-document>
  </xsl:template>
  
  <!-- Test of xsl:on-empty with xsl:if -->
  
  <xsl:template name="s-032" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
          <xsl:if test="current-date() lt xs:date('1900-01-01')">
            <banana x="{count(//*)}"/>
          </xsl:if>
          <xsl:on-empty>Howdy!</xsl:on-empty>  
      </out>
    </xsl:source-document>
  </xsl:template> 
  
  <!-- Test of xsl:on-empty with xsl:if -->
  
  <xsl:template name="s-033" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
          <xsl:if test="current-date() gt xs:date('1900-01-01')">
            <banana x="{count(//*)}"/>
          </xsl:if>
          <xsl:on-empty>Howdy!</xsl:on-empty>       
      </out>
    </xsl:source-document>
  </xsl:template> 
  
  <!-- Test of xsl:on-empty with xsl:for-each and xsl:where-populated -->
  
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
        <xsl:on-empty>There has been no gossip.</xsl:on-empty>
      </out>
    </xsl:source-document>
  </xsl:template> 
  
  <!-- Test of xsl:on-empty with xsl:fork -->
  
  <xsl:template name="s-037" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <a>
          <xsl:fork>
            <xsl:sequence select="copy-of(//TITLE)"/>
            <xsl:sequence select="copy-of(//PRICE)"/>  
          </xsl:fork>
          <xsl:on-empty>The forks were empty.</xsl:on-empty>
        </a>        
      </out>
    </xsl:source-document>
  </xsl:template> 
  
  <!-- Test of xsl:on-empty with xsl:fork -->
  
  <xsl:template name="s-038" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <a>
          <xsl:fork>
            <xsl:sequence select="copy-of(//TITTLE)"/>
            <xsl:sequence select="copy-of(//PRICLE)"/>  
          </xsl:fork>
          <xsl:on-empty>The forks were empty.</xsl:on-empty>
        </a>        
      </out>
    </xsl:source-document>
  </xsl:template> 
  
  <!-- Test of consuming xsl:on-empty (executed) -->
  
  <xsl:template name="s-039" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:if test="current-date() lt xs:date('1900-01-01')">
          <content/>
        </xsl:if>
        <xsl:on-empty>
          <xsl:for-each select="/BOOKLIST/BOOKS/ITEM">
            <xsl:copy-of select="TITLE"/>
          </xsl:for-each>
        </xsl:on-empty>
      </out>
    </xsl:source-document>
  </xsl:template> 
  
  <!-- Test of consuming xsl:on-empty (not executed) -->
  
  <xsl:template name="s-040" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:if test="current-date() gt xs:date('1900-01-01')">
          <content/>
        </xsl:if>
        <xsl:on-empty>
          <xsl:for-each select="/BOOKLIST/BOOKS/ITEM">
            <xsl:copy-of select="TITLE"/>
          </xsl:for-each>
        </xsl:on-empty>
      </out>
    </xsl:source-document>
  </xsl:template> 
  
  <!-- Zero length text nodes are considered empty -->
  
  <xsl:template name="s-041" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:value-of select="string(/BOOKLIST/BOOKS/ITEM/EXPORT-QUOTA)"/>
        <xsl:on-empty>EMPTY</xsl:on-empty>
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
        <xsl:on-empty>EMPTY</xsl:on-empty>
     </out>
    </xsl:source-document>
  </xsl:template> 
  
  <!-- Document nodes containing whitespace text are not considered empty -->
  
  <xsl:template name="s-043" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:document>
          <xsl:value-of select="concat(' ', string(/BOOKLIST/BOOKS/ITEM/EXPORT-QUOTA), ' ')"/>
        </xsl:document>  
        <xsl:on-empty>EMPTY</xsl:on-empty>
     </out>
    </xsl:source-document>
  </xsl:template> 
  
  




</xsl:stylesheet>
